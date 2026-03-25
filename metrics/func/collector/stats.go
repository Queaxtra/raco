package collector

import (
	"sort"
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

func CalculateStats(metrics []RequestMetric) Stats {
	stats := Stats{
		LastUpdated: time.Now(),
	}

	if len(metrics) == 0 {
		return stats
	}

	stats.TotalRequests = len(metrics)
	totalDuration := time.Duration(0)
	durations := make([]time.Duration, 0, len(metrics))
	stats.ProtocolCounts = make(map[string]int)

	for i, m := range metrics {
		if m.Success {
			stats.SuccessCount++
		}
		if !m.Success {
			stats.FailureCount++
		}

		totalDuration += m.Duration
		durations = append(durations, m.Duration)
		stats.ProtocolCounts[m.Protocol]++

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
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	stats.P50Duration = percentileDuration(durations, 50)
	stats.P95Duration = percentileDuration(durations, 95)
	stats.P99Duration = percentileDuration(durations, 99)

	return stats
}

func percentileDuration(durations []time.Duration, percentile int) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	if percentile <= 0 {
		return durations[0]
	}
	index := (len(durations) - 1) * percentile / 100
	if index < 0 {
		index = 0
	}
	if index >= len(durations) {
		index = len(durations) - 1
	}
	return durations[index]
}
