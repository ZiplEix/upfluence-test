package aggregation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ZiplEix/upfluence-test/eventbus"
)

func TestNewAggregationService(t *testing.T) {
	bus := eventbus.New[Item](10)
	svc := New(bus)
	if svc == nil {
		t.Fatal("expected non-nil AggregationService")
	}
	if svc.bus != bus {
		t.Error("expected service to hold provided eventbus")
	}
}

func TestAnalyze_TimeoutSuccess(t *testing.T) {
	bus := eventbus.New[Item](10)
	svc := New(bus)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Publish items with various timestamps and metrics
	go func() {
		for bus.SubscriberCount() == 0 {
			time.Sleep(1 * time.Millisecond)
		}
		// Matching dimension "likes"
		bus.Publish(&YoutubeVideo{Timestamp: 100, Likes: 10})
		bus.Publish(&InstagramMedia{Timestamp: 50, Likes: 20})
		bus.Publish(&Article{Timestamp: 200, Likes: 30})
		// Item that doesn't have "likes" dimension (tweet has favorites, retweets)
		bus.Publish(&Tweet{Timestamp: 150, Favorites: 999})
	}()

	report, err := svc.Analyze(ctx, "likes")
	if err != nil {
		t.Fatalf("expected nil error on timeout, got %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}

	if report.TotalPost != 3 {
		t.Errorf("expected 3 posts with likes, got %d", report.TotalPost)
	}
	if report.MinimumTimestamp != 50 {
		t.Errorf("expected minimum timestamp 50, got %d", report.MinimumTimestamp)
	}
	if report.MaximumTimestamp != 200 {
		t.Errorf("expected maximum timestamp 200, got %d", report.MaximumTimestamp)
	}
	if report.P50 != 20 {
		t.Errorf("expected p50=20, got %d", report.P50)
	}
}

func TestAnalyze_ContextCanceled(t *testing.T) {
	bus := eventbus.New[Item](10)
	svc := New(bus)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	report, err := svc.Analyze(ctx, "likes")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if report != nil {
		t.Errorf("expected nil report on cancel, got %v", report)
	}
}

func TestAnalyze_MultipleDimensions(t *testing.T) {
	bus := eventbus.New[Item](20)
	svc := New(bus)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	go func() {
		for bus.SubscriberCount() == 0 {
			time.Sleep(1 * time.Millisecond)
		}
		for i := 1; i <= 10; i++ {
			bus.Publish(&Tweet{Timestamp: int64(100 + i), Retweets: i * 10, Favorites: i * 5})
		}
	}()

	report, err := svc.Analyze(ctx, "retweets")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalPost != 10 {
		t.Errorf("expected 10 posts, got %d", report.TotalPost)
	}
	// For 10 items: len-1 = 9.
	// P50: 9 * 0.50 = 4.5 -> idx 4 -> value 50
	if report.P50 != 50 {
		t.Errorf("expected P50=50, got %d", report.P50)
	}
	// P90: 9 * 0.90 = 8.1 -> idx 8 -> value 90
	if report.P90 != 90 {
		t.Errorf("expected P90=90, got %d", report.P90)
	}
}

func TestAnalyze_EmptyItems(t *testing.T) {
	bus := eventbus.New[Item](10)
	svc := New(bus)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	report, err := svc.Analyze(ctx, "likes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalPost != 0 {
		t.Errorf("expected 0 total posts, got %d", report.TotalPost)
	}
	if report.MinimumTimestamp != 0 {
		t.Errorf("expected 0 min timestamp, got %d", report.MinimumTimestamp)
	}
	if report.MaximumTimestamp != 0 {
		t.Errorf("expected 0 max timestamp, got %d", report.MaximumTimestamp)
	}
	if report.P50 != 0 || report.P90 != 0 || report.P99 != 0 {
		t.Errorf("expected percentiles to be 0, got p50=%d p90=%d p99=%d", report.P50, report.P90, report.P99)
	}
}

func TestBuildReport_And_Percentile(t *testing.T) {
	t.Run("Empty slice", func(t *testing.T) {
		r := buildReport(0, 0, 0, nil, "likes")
		if r.TotalPost != 0 || r.P50 != 0 || r.P90 != 0 || r.P99 != 0 {
			t.Errorf("unexpected report for empty: %+v", r)
		}
	})

	t.Run("Single element", func(t *testing.T) {
		r := buildReport(1, 100, 100, []int{42}, "likes")
		if r.P50 != 42 || r.P90 != 42 || r.P99 != 42 {
			t.Errorf("expected all percentiles to be 42, got p50=%d p90=%d p99=%d", r.P50, r.P90, r.P99)
		}
	})

	t.Run("Multiple sorted elements", func(t *testing.T) {
		// 100 elements: 1, 2, ..., 100
		values := make([]int, 100)
		for i := 0; i < 100; i++ {
			values[i] = 100 - i // unsorted to test sort.Ints inside buildReport
		}

		r := buildReport(100, 10, 500, values, "likes")
		if r.TotalPost != 100 {
			t.Errorf("expected TotalPost 100, got %d", r.TotalPost)
		}
		// idx for 100 items: (100 - 1) * 0.50 = 49.5 -> 49 (value 50)
		if r.P50 != 50 {
			t.Errorf("expected P50=50, got %d", r.P50)
		}
		// idx for 100 items: 99 * 0.90 = 89.1 -> 89 (value 90)
		if r.P90 != 90 {
			t.Errorf("expected P90=90, got %d", r.P90)
		}
		// idx for 100 items: 99 * 0.99 = 98.01 -> 98 (value 99)
		if r.P99 != 99 {
			t.Errorf("expected P99=99, got %d", r.P99)
		}
	})

	t.Run("Percentile helper edge cases", func(t *testing.T) {
		if p := percentile(nil, 0.5); p != 0 {
			t.Errorf("expected 0 for nil slice, got %d", p)
		}
		if p := percentile([]int{}, 0.5); p != 0 {
			t.Errorf("expected 0 for empty slice, got %d", p)
		}
	})
}

