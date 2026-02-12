package model

import "time"

type StreamRequest struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Protocol string `json:"protocol" yaml:"protocol"`
	URL      string `json:"url" yaml:"url"`
	Message  string `json:"message" yaml:"message"`
}

type StreamMessage struct {
	Type      string    `json:"type"`
	Data      string    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
	Direction string    `json:"direction"`
}
