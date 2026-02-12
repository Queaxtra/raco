package message

import "time"

type Message struct {
	Type      string
	Data      string
	Timestamp time.Time
	Direction string
}
