package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type HistoryEntry struct {
	ID        string            `json:"id" yaml:"id"`
	Method    string            `json:"method" yaml:"method"`
	URL       string            `json:"url" yaml:"url"`
	Headers   map[string]string `json:"headers" yaml:"headers"`
	Body      string            `json:"body" yaml:"body"`
	Files     []FileUpload      `json:"files,omitempty" yaml:"files,omitempty"`
	Protocol  string            `json:"protocol" yaml:"protocol"`
	Timestamp time.Time         `json:"timestamp" yaml:"timestamp"`
}

func generateHistoryID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func NewHistoryEntry(method, url string, headers map[string]string, body, protocol string, files ...[]FileUpload) *HistoryEntry {
	entry := &HistoryEntry{
		ID:        generateHistoryID(),
		Method:    method,
		URL:       url,
		Headers:   headers,
		Body:      body,
		Protocol:  protocol,
		Timestamp: time.Now(),
	}
	if len(files) > 0 {
		entry.Files = files[0]
	}
	return entry
}
