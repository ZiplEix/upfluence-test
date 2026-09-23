package server

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingMiddleware_WithProvidedRequestID(t *testing.T) {
	handler := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "custom-request-id-1234")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("X-Request-ID"); got != "custom-request-id-1234" {
		t.Errorf("expected X-Request-ID custom-request-id-1234, got %s", got)
	}
}

func TestLoggingMiddleware_GeneratesRequestID(t *testing.T) {
	handler := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", rec.Code)
	}
	reqID := rec.Header().Get("X-Request-ID")
	if reqID == "" {
		t.Fatal("expected generated X-Request-ID, got empty")
	}
	if len(reqID) != 16 {
		t.Errorf("expected 16 hex chars, got %d (%s)", len(reqID), reqID)
	}
	if _, err := hex.DecodeString(reqID); err != nil {
		t.Errorf("expected valid hex string, got error: %v", err)
	}
}

func TestResponseWriterWrapper_StatusCodes(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	wrapper.WriteHeader(http.StatusNotFound)
	if wrapper.statusCode != http.StatusNotFound {
		t.Errorf("expected wrapper statusCode 404, got %d", wrapper.statusCode)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected underlying recorder code 404, got %d", rec.Code)
	}
}

func TestGenerateRequestID(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id := generateRequestID()
		if len(id) != 16 {
			t.Fatalf("expected length 16, got %d for %s", len(id), id)
		}
		if ids[id] {
			t.Fatalf("collision detected for id %s", id)
		}
		ids[id] = true
	}
}
