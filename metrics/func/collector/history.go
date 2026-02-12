package collector

import (
	"sync"
	"time"
)

type RequestMetric struct {
	Timestamp  time.Time
	Duration   time.Duration
	StatusCode int
	Success    bool
	Protocol   string
}

type History struct {
	items   []RequestMetric
	maxSize int
	mu      sync.RWMutex
}

func NewHistory(maxSize int) *History {
	return &History{
		items:   make([]RequestMetric, 0, maxSize),
		maxSize: maxSize,
	}
}

func (h *History) Add(metric RequestMetric) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.items = append(h.items, metric)

	if len(h.items) > h.maxSize {
		h.items = h.items[1:]
	}
}

func (h *History) GetRecent(count int) []RequestMetric {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if count > len(h.items) {
		count = len(h.items)
	}

	if count == 0 {
		return []RequestMetric{}
	}

	start := len(h.items) - count
	result := make([]RequestMetric, count)
	copy(result, h.items[start:])

	return result
}

func (h *History) GetAll() []RequestMetric {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]RequestMetric, len(h.items))
	copy(result, h.items)

	return result
}

func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.items = make([]RequestMetric, 0, h.maxSize)
}
