package metrics

import (
	"raco/metrics/func/collector"
)

type RequestMetric = collector.RequestMetric

type Collector struct {
	history *collector.History
}

func NewCollector(maxSize int) *Collector {
	return &Collector{
		history: collector.NewHistory(maxSize),
	}
}

func (c *Collector) Record(metric RequestMetric) {
	c.history.Add(metric)
}

func (c *Collector) GetRecent(count int) []RequestMetric {
	return c.history.GetRecent(count)
}

func (c *Collector) GetStats() Stats {
	raw := collector.CalculateStats(c.history.GetAll())
	return Stats{
		TotalRequests:   raw.TotalRequests,
		SuccessCount:    raw.SuccessCount,
		FailureCount:    raw.FailureCount,
		SuccessRate:     raw.SuccessRate,
		AverageDuration: raw.AverageDuration,
		MinDuration:     raw.MinDuration,
		MaxDuration:     raw.MaxDuration,
		LastUpdated:     raw.LastUpdated,
	}
}
