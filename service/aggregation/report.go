package aggregation

import (
	"encoding/json"
	"strings"
)

// Report contains aggregated metrics and calculated percentiles for a given dimension.
type Report struct {
	TotalPost        uint64 `json:"total_posts"`
	MinimumTimestamp int64  `json:"minimum_timestamp"`
	MaximumTimestamp int64  `json:"maximum_timestamp"`
	Dimension        string `json:"dimension,omitempty"`
	P50              int    `json:"p50"`
	P90              int    `json:"p90"`
	P99              int    `json:"p99"`
}

// MarshalJSON customizes JSON serialization so that percentiles are dynamically named
// after the analyzed dimension (e.g. likes_p50, likes_p90, likes_p99) matching the API specification.
func (r Report) MarshalJSON() ([]byte, error) {
	p50Key := "p50"
	p90Key := "p90"
	p99Key := "p99"
	if r.Dimension != "" {
		p50Key = r.Dimension + "_p50"
		p90Key = r.Dimension + "_p90"
		p99Key = r.Dimension + "_p99"
	}

	payload := map[string]any{
		"total_posts":       r.TotalPost,
		"minimum_timestamp": r.MinimumTimestamp,
		"maximum_timestamp": r.MaximumTimestamp,
		p50Key:              r.P50,
		p90Key:              r.P90,
		p99Key:              r.P99,
	}

	return json.Marshal(payload)
}

// UnmarshalJSON parses a Report from JSON, supporting both dynamic dimension-prefixed
// keys (e.g. "likes_p50") and standard static keys ("p50").
func (r *Report) UnmarshalJSON(data []byte) error {
	var raw struct {
		TotalPost        uint64 `json:"total_posts"`
		MinimumTimestamp int64  `json:"minimum_timestamp"`
		MaximumTimestamp int64  `json:"maximum_timestamp"`
		Dimension        string `json:"dimension"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.TotalPost = raw.TotalPost
	r.MinimumTimestamp = raw.MinimumTimestamp
	r.MaximumTimestamp = raw.MaximumTimestamp
	r.Dimension = raw.Dimension

	var dyn map[string]json.RawMessage
	if err := json.Unmarshal(data, &dyn); err != nil {
		return err
	}

	for k, v := range dyn {
		var val int
		if strings.HasSuffix(k, "_p50") || k == "p50" {
			if err := json.Unmarshal(v, &val); err == nil {
				r.P50 = val
				if strings.HasSuffix(k, "_p50") && r.Dimension == "" {
					r.Dimension = strings.TrimSuffix(k, "_p50")
				}
			}
		} else if strings.HasSuffix(k, "_p90") || k == "p90" {
			if err := json.Unmarshal(v, &val); err == nil {
				r.P90 = val
			}
		} else if strings.HasSuffix(k, "_p99") || k == "p99" {
			if err := json.Unmarshal(v, &val); err == nil {
				r.P99 = val
			}
		}
	}

	return nil
}
