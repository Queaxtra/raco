package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"raco/protocol/message"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

const (
	connectTimeout     = 10 * time.Second
	keepaliveTime      = 30 * time.Second
	keepaliveTimeout   = 10 * time.Second
	maxSendMsgSize     = 4 << 20  // 4MB
	maxRecvMsgSize     = 4 << 20  // 4MB
	defaultCallTimeout = 30 * time.Second
)

type Client struct {
	address       string
	conn          *grpc.ClientConn
	connected     atomic.Bool
	mu            sync.RWMutex
	messages      chan message.Message
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	closeOnce     sync.Once
	dialOptions   []grpc.DialOption
	insecureMode  bool
	currentStream *activeStream
}

type activeStream struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type CallRequest struct {
	Service  string          `json:"service"`
	Method   string          `json:"method"`
	Type     string          `json:"type"`
	Payload  interface{}     `json:"payload"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func NewClient(address string) *Client {
	return &Client{
		address:      address,
		insecureMode: false,
	}
}

func (c *Client) SetInsecure(insecure bool) {
	c.insecureMode = insecure
}

func (c *Client) Connect(ctx context.Context) error {
	if c.connected.Load() {
		return errors.New("already connected")
	}

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	creds := credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS12,
	})
	
	if c.insecureMode {
		creds = insecure.NewCredentials()
		c.safeLog("warning", "using insecure connection (TLS disabled)")
	}

	kp := keepalive.ClientParameters{
		Time:                keepaliveTime,
		Timeout:             keepaliveTimeout,
		PermitWithoutStream: true,
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(kp),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(maxSendMsgSize),
			grpc.MaxCallRecvMsgSize(maxRecvMsgSize),
		),
	}

	conn, err := grpc.DialContext(connectCtx, c.address, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected.Store(true)
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.messages = make(chan message.Message, 100)
	c.closeOnce = sync.Once{}
	c.mu.Unlock()

	c.safeLog("system", "gRPC connection established")
	
	c.wg.Add(1)
	go c.monitorConnection()

	return nil
}

func (c *Client) Send(data string) error {
	if !c.connected.Load() {
		return errors.New("not connected")
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return errors.New("connection is nil")
	}

	c.safeLog("sent", "message sent")
	c.safeLog("system", "gRPC Send requires JSON envelope with service/method/type")

	return nil
}

func (c *Client) SendMessage(service, method string, payload []byte, md map[string]string) error {
	if !c.connected.Load() {
		return errors.New("not connected")
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return errors.New("connection is nil")
	}

	ctx := metadataContext(c.ctx, md)
	callCtx, cancel := context.WithTimeout(ctx, defaultCallTimeout)
	defer cancel()

	fullMethod := fmt.Sprintf("/%s/%s", service, method)

	var respBytes []byte
	err := conn.Invoke(callCtx, fullMethod, payload, &respBytes)

	if err != nil {
		c.safeLog("error", fmt.Sprintf("RPC failed: %v", err))
		return err
	}

	c.safeLog("received", fmt.Sprintf("response received (%d bytes)", len(respBytes)))
	return nil
}

func (c *Client) Receive() (<-chan message.Message, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected.Load() {
		return nil, errors.New("not connected")
	}

	return c.messages, nil
}

func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.connected.Store(false)

		if c.cancel != nil {
			c.cancel()
		}

		c.mu.Lock()
		if c.currentStream != nil && c.currentStream.cancel != nil {
			c.currentStream.cancel()
			c.currentStream.wg.Wait()
		}
		conn := c.conn
		c.conn = nil
		c.mu.Unlock()

		if conn != nil {
			conn.Close()
		}

		c.wg.Wait()

		c.mu.Lock()
		if c.messages != nil {
			close(c.messages)
			c.messages = nil
		}
		c.mu.Unlock()
	})

	return nil
}

func (c *Client) IsConnected() bool {
	if !c.connected.Load() {
		return false
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return false
	}

	state := conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

func (c *Client) ConnectionState() string {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return "Disconnected"
	}

	return conn.GetState().String()
}

func (c *Client) monitorConnection() {
	defer c.wg.Done()

	c.mu.RLock()
	conn := c.conn
	ctx := c.ctx
	c.mu.RUnlock()

	if conn == nil {
		return
	}

	state := conn.GetState()
	conn.Connect()

	for {
		if !conn.WaitForStateChange(ctx, state) {
			return
		}
		state = conn.GetState()
		
		switch state {
		case connectivity.TransientFailure:
			c.safeLog("error", "connection transient failure")
		case connectivity.Shutdown:
			c.safeLog("system", "connection shutdown")
			c.connected.Store(false)
			return
		case connectivity.Ready:
			c.safeLog("system", "connection ready")
		}
	}
}

func (c *Client) safeLog(msgType, data string) {
	c.mu.RLock()
	msgCh := c.messages
	connected := c.connected.Load()
	c.mu.RUnlock()

	if !connected || msgCh == nil {
		return
	}

	select {
	case msgCh <- message.Message{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		Direction: "system",
	}:
	default:
	}
}

func metadataContext(ctx context.Context, md map[string]string) context.Context {
	if len(md) == 0 {
		return ctx
	}

	pairs := make([]string, 0, len(md)*2)
	for k, v := range md {
		pairs = append(pairs, k, v)
	}

	return metadata.AppendToOutgoingContext(ctx, pairs...)
}
