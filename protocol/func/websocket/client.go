package websocket

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"raco/protocol/message"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxMessageSize   = 1 << 20 // 1MB max message size
	handshakeTimeout = 10 * time.Second
	pingPeriod       = 30 * time.Second
	pongWait         = 60 * time.Second
	writeTimeout     = 10 * time.Second
)

type Client struct {
	url          string
	conn         *websocket.Conn
	connected    atomic.Bool
	mu           sync.RWMutex
	messages     chan message.Message
	sendCh       chan sendPayload
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	closeOnce    sync.Once
	dialer       *websocket.Dialer
}

type sendPayload struct {
	data []byte
	done chan error
}

func NewClient(targetURL string) *Client {
	return &Client{
		url:    targetURL,
		dialer: defaultDialer(),
	}
}

func defaultDialer() *websocket.Dialer {
	return &websocket.Dialer{
		HandshakeTimeout: handshakeTimeout,
		Proxy:            http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
}

func (c *Client) Connect(ctx context.Context) error {
	if c.connected.Load() {
		return errors.New("already connected")
	}

	parsedURL, err := url.Parse(c.url)
	if err != nil {
		return errors.New("invalid URL: " + err.Error())
	}

	if parsedURL.Scheme != "ws" && parsedURL.Scheme != "wss" {
		return errors.New("URL must use ws:// or wss:// scheme")
	}

	connectCtx, cancel := context.WithTimeout(ctx, handshakeTimeout)
	defer cancel()

	conn, resp, err := c.dialer.DialContext(connectCtx, c.url, nil)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return err
	}

	conn.SetReadLimit(maxMessageSize)
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	conn.SetReadDeadline(time.Now().Add(pongWait))

	c.mu.Lock()
	c.conn = conn
	c.connected.Store(true)
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.messages = make(chan message.Message, 100)
	c.sendCh = make(chan sendPayload, 10)
	c.closeOnce = sync.Once{}
	c.mu.Unlock()

	c.wg.Add(2)
	go c.readLoop()
	go c.writeLoop()

	if parsedURL.Scheme == "ws" {
		c.safeSendMessage(c.messages, message.Message{
			Type:      "warning",
			Data:      "using unencrypted ws:// connection",
			Timestamp: time.Now(),
			Direction: "system",
		})
	}

	return nil
}

func (c *Client) Send(data string) error {
	if !c.connected.Load() {
		return errors.New("not connected")
	}

	payload := sendPayload{
		data: []byte(data),
		done: make(chan error, 1),
	}

	select {
	case c.sendCh <- payload:
		err := <-payload.done
		if err != nil {
			return err
		}
		
		c.sendToBuffer(data)
		return nil
	case <-c.ctx.Done():
		return errors.New("connection closed")
	case <-time.After(writeTimeout):
		return errors.New("send timeout")
	}
}

func (c *Client) sendToBuffer(data string) {
	select {
	case c.messages <- message.Message{
		Type:      "text",
		Data:      data,
		Timestamp: time.Now(),
		Direction: "sent",
	}:
	default:
		select {
		case c.messages <- message.Message{
			Type:      "system",
			Data:      "message buffer full - sent message dropped from log",
			Timestamp: time.Now(),
			Direction: "system",
		}:
		default:
		}
	}
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
	var closeErr error

	c.closeOnce.Do(func() {
		if !c.connected.Load() {
			return
		}

		c.connected.Store(false)

		if c.cancel != nil {
			c.cancel()
		}

		c.mu.Lock()
		conn := c.conn
		c.conn = nil
		c.mu.Unlock()

		if conn != nil {
			closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client disconnect")
			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			conn.WriteMessage(websocket.CloseMessage, closeMsg)
			conn.Close()
		}

		close(c.sendCh)
		c.wg.Wait()

		c.mu.Lock()
		if c.messages != nil {
			close(c.messages)
			c.messages = nil
		}
		c.mu.Unlock()
	})

	return closeErr
}

func (c *Client) IsConnected() bool {
	return c.connected.Load()
}

func (c *Client) readLoop() {
	defer c.wg.Done()

	c.mu.RLock()
	conn := c.conn
	msgCh := c.messages
	ctx := c.ctx
	c.mu.RUnlock()

	if conn == nil {
		return
	}

	defer func() {
		c.mu.Lock()
		c.connected.Store(false)
		conn := c.conn
		c.conn = nil
		c.mu.Unlock()

		if conn != nil {
			closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "read loop exit")
			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			conn.WriteMessage(websocket.CloseMessage, closeMsg)
			conn.Close()
		}

		if c.cancel != nil {
			c.cancel()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(pongWait))
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				c.safeSendError(msgCh, "unexpected close: "+err.Error())
			} else if ctx.Err() == nil {
				c.safeSendError(msgCh, "read error: "+err.Error())
			}
			return
		}

		msgTypeStr := "text"
		if messageType == websocket.BinaryMessage {
			msgTypeStr = "binary"
		}

		c.safeSendMessage(msgCh, message.Message{
			Type:      msgTypeStr,
			Data:      string(data),
			Timestamp: time.Now(),
			Direction: "received",
		})
	}
}

func (c *Client) writeLoop() {
	defer c.wg.Done()

	c.mu.RLock()
	conn := c.conn
	ctx := c.ctx
	c.mu.RUnlock()

	if conn == nil {
		return
	}

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case payload, ok := <-c.sendCh:
			if !ok {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			err := conn.WriteMessage(websocket.TextMessage, payload.data)
			payload.done <- err
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) safeSendMessage(ch chan message.Message, msg message.Message) {
	select {
	case ch <- msg:
	default:
	}
}

func (c *Client) safeSendError(ch chan message.Message, errStr string) {
	c.safeSendMessage(ch, message.Message{
		Type:      "error",
		Data:      errStr,
		Timestamp: time.Now(),
		Direction: "system",
	})
}
