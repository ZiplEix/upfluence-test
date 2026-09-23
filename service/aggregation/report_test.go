package aggregation

import (
	"encoding/json"
	"testing"
)

func TestReport_MarshalJSON(t *testing.T) {
	report := &Report{
		TotalPost:        20,
		MinimumTimestamp: 1000,
		MaximumTimestamp: 2000,
		Dimension:        "likes",
		P50:              50,
		P90:              90,
		P99:              99,
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal JSON map: %v", err)
	}

	if _, ok := m["dimension"]; ok {
		t.Errorf("expected no 'dimension' field in JSON, got %v", m["dimension"])
	}

	if m["likes_p50"] != float64(50) {
		t.Errorf("expected likes_p50=50, got %v", m["likes_p50"])
	}
	if m["likes_p90"] != float64(90) {
		t.Errorf("expected likes_p90=90, got %v", m["likes_p90"])
	}
	if m["likes_p99"] != float64(99) {
		t.Errorf("expected likes_p99=99, got %v", m["likes_p99"])
	}
	if m["total_posts"] != float64(20) {
		t.Errorf("expected total_posts=20, got %v", m["total_posts"])
	}
}

func TestReport_UnmarshalJSON(t *testing.T) {
	jsonInput := []byte(`{
		"total_posts": 25,
		"minimum_timestamp": 500,
		"maximum_timestamp": 1500,
		"comments_p50": 12,
		"comments_p90": 45,
		"comments_p99": 88
	}`)

	var r Report
	if err := json.Unmarshal(jsonInput, &r); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if r.TotalPost != 25 {
		t.Errorf("expected TotalPost=25, got %d", r.TotalPost)
	}
	if r.MinimumTimestamp != 500 || r.MaximumTimestamp != 1500 {
		t.Errorf("unexpected timestamps: min=%d max=%d", r.MinimumTimestamp, r.MaximumTimestamp)
	}
	if r.Dimension != "comments" {
		t.Errorf("expected Dimension='comments', got '%s'", r.Dimension)
	}
	if r.P50 != 12 || r.P90 != 45 || r.P99 != 88 {
		t.Errorf("unexpected percentiles: p50=%d p90=%d p99=%d", r.P50, r.P90, r.P99)
	}
}
