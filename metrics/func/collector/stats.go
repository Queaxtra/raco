package collector

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

func CalculateStats(metrics []RequestMetric) Stats {
	stats := Stats{
		LastUpdated: time.Now(),
	}

	if len(metrics) == 0 {
		return stats
	}

	stats.TotalRequests = len(metrics)
	totalDuration := time.Duration(0)

	for i, m := range metrics {
		if m.Success {
			stats.SuccessCount++
		}
		if !m.Success {
			stats.FailureCount++
		}

		totalDuration += m.Duration

		if i == 0 {
			stats.MinDuration = m.Duration
			stats.MaxDuration = m.Duration
		}

		if m.Duration < stats.MinDuration {
			stats.MinDuration = m.Duration
		}
		if m.Duration > stats.MaxDuration {
			stats.MaxDuration = m.Duration
		}
	}

	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalRequests) * 100
		stats.AverageDuration = totalDuration / time.Duration(stats.TotalRequests)
	}

	return stats
}
