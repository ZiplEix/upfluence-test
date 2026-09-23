package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZiplEix/upfluence-test/eventbus"
	"github.com/ZiplEix/upfluence-test/service/aggregation"
)

func TestHandlerAggregation_InvalidDuration(t *testing.T) {
	bus := eventbus.New[aggregation.Item](10)
	svc := aggregation.New(bus)
	api := New(WithAggregation(svc))

	req := httptest.NewRequest(http.MethodGet, "/analysis?dimension=likes&duration=invalid-time", nil)
	rec := httptest.NewRecorder()

	api.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid duration parameter") {
		t.Errorf("unexpected error body: %s", rec.Body.String())
	}
}

func TestHandlerAggregation_NegativeDuration(t *testing.T) {
	bus := eventbus.New[aggregation.Item](10)
	svc := aggregation.New(bus)
	api := New(WithAggregation(svc))

	req := httptest.NewRequest(http.MethodGet, "/analysis?dimension=likes&duration=-5s", nil)
	rec := httptest.NewRecorder()

	api.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid duration parameter, duration must be > 0") {
		t.Errorf("unexpected error body: %s", rec.Body.String())
	}
}

func TestHandlerAggregation_InvalidOrMissingDimension(t *testing.T) {
	bus := eventbus.New[aggregation.Item](10)
	svc := aggregation.New(bus)
	api := New(WithAggregation(svc))

	// Missing dimension
	reqMissing := httptest.NewRequest(http.MethodGet, "/analysis?duration=50ms", nil)
	recMissing := httptest.NewRecorder()
	api.mux.ServeHTTP(recMissing, reqMissing)

	if recMissing.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", recMissing.Code)
	}

	// Invalid dimension
	reqInvalid := httptest.NewRequest(http.MethodGet, "/analysis?duration=50ms&dimension=unsupported", nil)
	recInvalid := httptest.NewRecorder()
	api.mux.ServeHTTP(recInvalid, reqInvalid)

	if recInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", recInvalid.Code)
	}
	if !strings.Contains(recInvalid.Body.String(), "invalid or missing dimension parameter") {
		t.Errorf("unexpected error body: %s", recInvalid.Body.String())
	}
}

func TestHandlerAggregation_SuccessAllDimensions(t *testing.T) {
	dimensions := []string{"likes", "comments", "favorites", "retweets"}

	for _, dim := range dimensions {
		t.Run(dim, func(t *testing.T) {
			bus := eventbus.New[aggregation.Item](10)
			svc := aggregation.New(bus)
			api := New(WithAggregation(svc))

			req := httptest.NewRequest(http.MethodGet, "/analysis?dimension="+dim+"&duration=30ms", nil)
			rec := httptest.NewRecorder()

			go func() {
				for bus.SubscriberCount() == 0 {
					time.Sleep(1 * time.Millisecond)
				}
				// Publish items
				bus.Publish(&aggregation.Tweet{Timestamp: 1000, Favorites: 10, Retweets: 5})
				bus.Publish(&aggregation.YoutubeVideo{Timestamp: 2000, Likes: 50, Comments: 12})
			}()

			api.mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
			}

			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", ct)
			}

			var report aggregation.Report
			err := json.Unmarshal(rec.Body.Bytes(), &report)
			if err != nil {
				t.Fatalf("failed to parse JSON response: %v", err)
			}

			if report.TotalPost < 1 {
				t.Errorf("expected at least 1 post for dimension %s, got %d", dim, report.TotalPost)
			}
		})
	}
}

func TestHandlerAggregation_WithoutDuration_CanceledContext(t *testing.T) {
	bus := eventbus.New[aggregation.Item](10)
	svc := aggregation.New(bus)
	api := New(WithAggregation(svc))

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/analysis?dimension=likes", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	go func() {
		// Cancel shortly after starting to test infinite duration and context.Canceled handling
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	api.mux.ServeHTTP(rec, req)

	// Since context was canceled, the handler should return without writing a 500 error
	if rec.Code == http.StatusInternalServerError {
		t.Errorf("did not expect 500 status on client cancel, got %d", rec.Code)
	}
}
