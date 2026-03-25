package metrics

import (
	"time"
)

type Stats struct {
	TotalRequests   int
	SuccessCount    int
	FailureCount    int
	SuccessRate     float64
	AverageDuration time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	P50Duration     time.Duration
	P95Duration     time.Duration
	P99Duration     time.Duration
	ProtocolCounts  map[string]int
	LastUpdated     time.Time
}
