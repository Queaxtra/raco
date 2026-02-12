package protocol

import (
	"raco/protocol/func/websocket"
)

func NewWebSocketClient(url string) StreamHandler {
	return websocket.NewClient(url)
}
