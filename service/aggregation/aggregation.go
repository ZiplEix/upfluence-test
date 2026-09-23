// Package aggregation provides real-time event aggregation and statistical reporting
// based on incoming items from an event bus.
package aggregation

import (
	"context"
	"errors"
	"sort"

	"github.com/ZiplEix/upfluence-test/eventbus"
)

// AggregationService is the service running the aggregation computation and logic
type AggregationService struct {
	bus *eventbus.EventBus[Item]
}

// New initializes and returns a new AggregationService listening to the provided event bus.
func New(bus *eventbus.EventBus[Item]) *AggregationService {
	return &AggregationService{
		bus: bus,
	}
}

// Analyze subscribes to the event bus and aggregates metrics for the specified dimension
// until the context deadline expires or the channel closes.
// Returns a computed Report, or an error if the context is canceled unexpectedly.
func (s *AggregationService) Analyze(ctx context.Context, dimension string) (*Report, error) {
	eventsCh, unsubscribe := s.bus.Subscribe()
	defer unsubscribe()

	var (
		totalPosts uint64
		minTime    int64
		maxTime    int64
		values     []int
	)

	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return buildReport(totalPosts, minTime, maxTime, values, dimension), nil
			}
			return nil, ctx.Err()

		case item, ok := <-eventsCh:
			if !ok {
				return buildReport(totalPosts, minTime, maxTime, values, dimension), nil
			}

			val, exists := item.Metric(dimension)
			if !exists {
				continue
			}

			totalPosts++

			ts := item.OccurredAt()
			if minTime == 0 || ts < minTime {
				minTime = ts
			}
			if ts > maxTime {
				maxTime = ts
			}

			values = append(values, val)
		}
	}
}

func buildReport(total uint64, minTime, maxTime int64, likes []int, dimension string) *Report {
	r := &Report{
		TotalPost:        total,
		MinimumTimestamp: minTime,
		MaximumTimestamp: maxTime,
		Dimension:        dimension,
	}

	if len(likes) == 0 {
		return r
	}

	sort.Ints(likes)
	r.P50 = percentile(likes, 0.50)
	r.P90 = percentile(likes, 0.90)
	r.P99 = percentile(likes, 0.99)
	return r
}

func percentile(sorted []int, p float64) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}
