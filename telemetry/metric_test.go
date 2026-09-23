package telemetry

import (
	"expvar"
	"testing"
)

func TestMetricsRegistration(t *testing.T) {
	metrics := []struct {
		name string
		val  expvar.Var
	}{
		{"eventbus_active_subscribers", ActiveSubscribers},
		{"eventbus_dropped_events_total", DroppedEvents},
		{"stream_worker_connected", StreamStatus},
		{"stream_worker_reconnections_total", StreamReconnections},
		{"stream_worker_errors_total", StreamErrors},
		{"stream_events_ingested_total", EventsIngested},
		{"http_requests_total", HTTPRequestsTotal},
		{"http_requests_by_status", HTTPRequestsByStatus},
		{"analysis_requests_by_dimension", AnalysisByDimension},
	}

	for _, m := range metrics {
		t.Run(m.name, func(t *testing.T) {
			if m.val == nil {
				t.Fatalf("metric %s is nil", m.name)
			}
			registered := expvar.Get(m.name)
			if registered == nil {
				t.Fatalf("metric %s is not registered in expvar", m.name)
			}
			if registered != m.val {
				t.Fatalf("metric %s does not match expvar registry", m.name)
			}
		})
	}
}

func TestMetricsOperations(t *testing.T) {
	StreamStatus.Set(1)
	if StreamStatus.Value() != 1 {
		t.Errorf("expected StreamStatus 1, got %d", StreamStatus.Value())
	}
	StreamStatus.Set(0)
	if StreamStatus.Value() != 0 {
		t.Errorf("expected StreamStatus 0, got %d", StreamStatus.Value())
	}

	before := HTTPRequestsTotal.Value()
	HTTPRequestsTotal.Add(1)
	if HTTPRequestsTotal.Value() != before+1 {
		t.Errorf("expected HTTPRequestsTotal to increment by 1")
	}

	EventsIngested.Add("tweet", 1)
	EventsIngested.Add("tweet", 2)
	v := EventsIngested.Get("tweet")
	if v == nil {
		t.Fatal("expected 'tweet' in EventsIngested")
	}
	intVal, ok := v.(*expvar.Int)
	if !ok || intVal.Value() < 3 {
		t.Errorf("expected >= 3 for tweet, got %v", v)
	}

	AnalysisByDimension.Add("likes", 1)
	vDim := AnalysisByDimension.Get("likes")
	if vDim == nil {
		t.Fatal("expected 'likes' in AnalysisByDimension")
	}
}
