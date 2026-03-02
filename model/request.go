package model

import "time"

type Request struct {
	ID             string            `json:"id" yaml:"id"`
	Name           string            `json:"name" yaml:"name"`
	Method         string            `json:"method" yaml:"method"`
	URL            string            `json:"url" yaml:"url"`
	Query          map[string]string  `json:"query,omitempty" yaml:"query,omitempty"`
	Headers        map[string]string `json:"headers" yaml:"headers"`
	Body           string            `json:"body" yaml:"body"`
	Files          []FileUpload      `json:"files,omitempty" yaml:"files,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
	CreatedAt      time.Time         `json:"created_at" yaml:"created_at"`
	CollectionID   string            `json:"collection_id" yaml:"collection_id"`
	Assertions     []Assertion       `json:"assertions,omitempty" yaml:"assertions,omitempty"`
	Extractors     []Extractor       `json:"extractors,omitempty" yaml:"extractors,omitempty"`
}

type Response struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Duration   time.Duration     `json:"duration"`
	Timestamp  time.Time         `json:"timestamp"`
}
