package server

import (
	"expvar"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ZiplEix/upfluence-test/telemetry"
)

func TestTelemetryMiddleware_RequestsAndStatus(t *testing.T) {
	handler := telemetryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	reqTotalBefore := telemetry.HTTPRequestsTotal.Value()

	req := httptest.NewRequest(http.MethodGet, "/teapot", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("expected status 418, got %d", rec.Code)
	}

	if telemetry.HTTPRequestsTotal.Value() != reqTotalBefore+1 {
		t.Errorf("expected HTTPRequestsTotal to increment by 1")
	}

	v := telemetry.HTTPRequestsByStatus.Get("418")
	if v == nil {
		t.Fatal("expected status 418 in HTTPRequestsByStatus map")
	}
	intVal, ok := v.(*expvar.Int)
	if !ok || intVal.Value() < 1 {
		t.Errorf("expected status 418 count >= 1, got %v", v)
	}
}
