package protocol

import (
	"context"
	"raco/protocol/message"
)

type ProtocolType string

const (
	ProtocolHTTP      ProtocolType = "HTTP"
	ProtocolWebSocket ProtocolType = "WebSocket"
	ProtocolGRPC      ProtocolType = "gRPC"
)

type Message = message.Message

type StreamHandler interface {
	Connect(ctx context.Context) error
	Send(data string) error
	Receive() (<-chan message.Message, error)
	Close() error
	IsConnected() bool
}
