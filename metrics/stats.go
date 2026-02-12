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
	LastUpdated     time.Time
}
