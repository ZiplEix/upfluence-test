package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/ZiplEix/upfluence-test/eventbus"
	"github.com/ZiplEix/upfluence-test/server"
	"github.com/ZiplEix/upfluence-test/service/aggregation"
)

// mockUpstreamStream spins up an SSE HTTP server that yields JSON stream events continuously.
func mockUpstreamStream(t *testing.T, stopCh <-chan struct{}) *httptest.Server {
	t.Helper()

	events := []string{
		`{"tweet":{"id":1001,"content":"hello","favorites":15,"retweets":5,"timestamp":1700000010}}`,
		`{"youtube_video":{"id":2001,"name":"video1","views":500,"likes":80,"comments":10,"timestamp":1700000020}}`,
		`{"instagram_media":{"id":3001,"text":"photo1","likes":120,"comments":25,"views":600,"timestamp":1700000030}}`,
		`{"facebook_status":{"id":4001,"status":"status1","likes":45,"comments":8,"timestamp":1700000040}}`,
		`{"pin":{"id":5001,"repins":20,"likes":60,"comments":12,"timestamp":1700000050}}`,
		`{"article":{"id":6001,"title":"article1","likes":95,"comments":18,"timestamp":1700000060}}`,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		flusher.Flush()

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-r.Context().Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				payload := events[i%len(events)]
				i++
				if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	})

	return httptest.NewServer(handler)
}

func setupE2EEnvironment(t *testing.T) (apiAddr string, cleanup func()) {
	t.Helper()

	stopStream := make(chan struct{})
	streamServer := mockUpstreamStream(t, stopStream)

	appCtx, cancelApp := context.WithCancel(context.Background())

	bus := eventbus.New[aggregation.Item](500)

	worker := aggregation.NewStreamWorker(streamServer.URL, bus)
	worker.Start(appCtx)

	aggregationService := aggregation.New(bus)

	// Allocate free dynamic port for server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate free port: %v", err)
	}
	serverAddr := ln.Addr().String()
	_ = ln.Close()

	httpSrv := &http.Server{
		Addr: serverAddr,
	}

	api := server.New(
		server.WithServer(httpSrv),
		server.WithAggregation(aggregationService),
	)

	go func() {
		_ = api.Serve()
	}()

	// Wait for server to be responsive
	for {
		conn, err := net.DialTimeout("tcp", serverAddr, 20*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	cleanup = func() {
		close(stopStream)
		streamServer.Close()
		cancelApp()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer shutdownCancel()
		_ = api.Shutdown(shutdownCtx)
	}

	return serverAddr, cleanup
}

func TestE2E_FullPipeline(t *testing.T) {
	serverAddr, cleanup := setupE2EEnvironment(t)
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}

	// 1. Query for "likes" dimension over 150ms window
	reqURL := fmt.Sprintf("http://%s/analysis?dimension=likes&duration=150ms", serverAddr)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-Request-ID", "e2e-likes-req-1")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to perform request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 OK, got %d: %s", resp.StatusCode, string(body))
	}

	if reqID := resp.Header.Get("X-Request-ID"); reqID != "e2e-likes-req-1" {
		t.Errorf("expected X-Request-ID header to match, got %s", reqID)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var rawMap map[string]any
	if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
		t.Fatalf("failed to unmarshal JSON map: %v", err)
	}
	if _, exists := rawMap["likes_p50"]; !exists {
		t.Errorf("expected 'likes_p50' key in response JSON, got %v", rawMap)
	}
	if _, exists := rawMap["dimension"]; exists {
		t.Errorf("expected no 'dimension' field in response JSON, got %v", rawMap["dimension"])
	}

	var report aggregation.Report
	if err := json.Unmarshal(bodyBytes, &report); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	if report.TotalPost == 0 {
		t.Fatal("expected total_posts > 0 in aggregated report")
	}
	if report.MinimumTimestamp == 0 || report.MaximumTimestamp == 0 {
		t.Errorf("expected timestamps to be non-zero, got min=%d max=%d", report.MinimumTimestamp, report.MaximumTimestamp)
	}
	if report.MaximumTimestamp < report.MinimumTimestamp {
		t.Errorf("maximum timestamp (%d) is less than minimum timestamp (%d)", report.MaximumTimestamp, report.MinimumTimestamp)
	}
	if report.P50 == 0 || report.P90 == 0 || report.P99 == 0 {
		t.Errorf("expected non-zero percentiles, got p50=%d p90=%d p99=%d", report.P50, report.P90, report.P99)
	}

	// 2. Query for "favorites" dimension (tweets only)
	favURL := fmt.Sprintf("http://%s/analysis?dimension=favorites&duration=150ms", serverAddr)
	respFav, err := client.Get(favURL)
	if err != nil {
		t.Fatalf("failed to query favorites: %v", err)
	}
	defer func() { _ = respFav.Body.Close() }()

	if respFav.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for favorites, got %d", respFav.StatusCode)
	}

	bodyFavBytes, err := io.ReadAll(respFav.Body)
	if err != nil {
		t.Fatalf("failed to read favorites body: %v", err)
	}

	var favRawMap map[string]any
	if err := json.Unmarshal(bodyFavBytes, &favRawMap); err != nil {
		t.Fatalf("failed to unmarshal favorites JSON map: %v", err)
	}
	if _, exists := favRawMap["favorites_p50"]; !exists {
		t.Errorf("expected 'favorites_p50' key in response JSON, got %v", favRawMap)
	}

	var favReport aggregation.Report
	if err := json.Unmarshal(bodyFavBytes, &favReport); err != nil {
		t.Fatalf("failed to decode favorites report: %v", err)
	}

	if favReport.TotalPost == 0 {
		t.Fatal("expected favorites total_posts > 0")
	}
	if favReport.P50 != 15 {
		t.Errorf("expected tweet favorites P50=15, got %d", favReport.P50)
	}
}

func TestE2E_ConcurrentClients(t *testing.T) {
	serverAddr, cleanup := setupE2EEnvironment(t)
	defer cleanup()

	const concurrency = 8
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	dimensions := []string{"likes", "comments", "favorites", "retweets"}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			dim := dimensions[idx%len(dimensions)]
			url := fmt.Sprintf("http://%s/analysis?dimension=%s&duration=100ms", serverAddr, dim)

			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				errCh <- fmt.Errorf("client %d: request creation failed: %w", idx, err)
				return
			}
			req.Header.Set("X-Request-ID", fmt.Sprintf("concurrent-client-%d", idx))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errCh <- fmt.Errorf("client %d: request failed: %w", idx, err)
				return
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				errCh <- fmt.Errorf("client %d: status code %d", idx, resp.StatusCode)
				return
			}

			var report aggregation.Report
			if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
				errCh <- fmt.Errorf("client %d: json decode failed: %w", idx, err)
				return
			}

			if report.TotalPost == 0 {
				errCh <- fmt.Errorf("client %d: expected total_posts > 0, got 0", idx)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent test failure: %v", err)
	}
}

func TestE2E_ValidationErrors(t *testing.T) {
	serverAddr, cleanup := setupE2EEnvironment(t)
	defer cleanup()

	client := &http.Client{Timeout: 3 * time.Second}

	cases := []struct {
		name         string
		url          string
		expectedCode int
	}{
		{
			name:         "Invalid duration string",
			url:          fmt.Sprintf("http://%s/analysis?dimension=likes&duration=bad_duration", serverAddr),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Negative duration parameter",
			url:          fmt.Sprintf("http://%s/analysis?dimension=likes&duration=-5s", serverAddr),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Missing dimension parameter",
			url:          fmt.Sprintf("http://%s/analysis?duration=50ms", serverAddr),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Unsupported dimension parameter",
			url:          fmt.Sprintf("http://%s/analysis?dimension=views&duration=50ms", serverAddr),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Wrong HTTP method",
			url:          fmt.Sprintf("http://%s/analysis?dimension=likes&duration=50ms", serverAddr),
			expectedCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			if tc.expectedCode == http.StatusMethodNotAllowed {
				resp, err = client.Post(tc.url, "application/json", nil)
			} else {
				resp, err = client.Get(tc.url)
			}

			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if tc.name == "Wrong HTTP method" {
				if resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotFound {
					t.Errorf("expected status 404 or 405 for wrong method, got %d", resp.StatusCode)
				}
				return
			}

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("expected status %d, got %d", tc.expectedCode, resp.StatusCode)
			}
		})
	}
}

func TestE2E_MainBinaryExecution(t *testing.T) {
	// Build the real binary to test full main() execution and signal handling
	tmpDir, err := os.MkdirTemp("", "upfluence-bin-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	binPath := filepath.Join(tmpDir, "upfluence-test-bin")

	// Compile binary
	buildCmd := exec.Command("go", "build", "-o", binPath, "main.go")
	buildCmd.Dir = ".."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\nOutput: %s", err, string(out))
	}

	// Run the compiled binary
	cmd := exec.Command(binPath)
	cmd.Dir = ".."

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start binary: %v", err)
	}

	// Give it a brief moment to start, then send SIGTERM for graceful shutdown
	time.Sleep(300 * time.Millisecond)

	// Send SIGTERM to test the signal.Notify and graceful shutdown logic in main.go
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("failed to send SIGTERM: %v", err)
	}

	// Wait for process to exit cleanly
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("process exited with error: %v", err)
		}
	case <-time.After(6 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("process did not shutdown within 6 seconds after SIGTERM")
	}
}

// func TestE2E_TelemetryDebugVars(t *testing.T) {
// 	serverAddr, cleanup := setupE2EEnvironment(t)
// 	defer cleanup()

// 	// 1. Perform an aggregation query to trigger metrics
// 	analysisURL := fmt.Sprintf("http://%s/analysis?dimension=likes&duration=80ms", serverAddr)
// 	resp, err := http.Get(analysisURL)
// 	if err != nil {
// 		t.Fatalf("failed to query analysis: %v", err)
// 	}
// 	_ = resp.Body.Close()
// 	if resp.StatusCode != http.StatusOK {
// 		t.Fatalf("expected 200 OK for analysis, got %d", resp.StatusCode)
// 	}

// 	// 2. Query /debug/vars
// 	debugURL := fmt.Sprintf("http://%s/debug/vars", serverAddr)
// 	debugResp, err := http.Get(debugURL)
// 	if err != nil {
// 		t.Fatalf("failed to query /debug/vars: %v", err)
// 	}
// 	defer func() { _ = debugResp.Body.Close() }()

// 	if debugResp.StatusCode != http.StatusOK {
// 		t.Fatalf("expected 200 OK from /debug/vars, got %d", debugResp.StatusCode)
// 	}

// 	var debugData map[string]any
// 	if err := json.NewDecoder(debugResp.Body).Decode(&debugData); err != nil {
// 		t.Fatalf("failed to decode /debug/vars JSON: %v", err)
// 	}

// 	// Verify required keys are present
// 	expectedKeys := []string{
// 		"eventbus_active_subscribers",
// 		"eventbus_dropped_events_total",
// 		"stream_worker_connected",
// 		"stream_worker_reconnections_total",
// 		"stream_worker_errors_total",
// 		"stream_events_ingested_total",
// 		"http_requests_total",
// 		"http_requests_by_status",
// 		"analysis_requests_by_dimension",
// 	}

// 	for _, key := range expectedKeys {
// 		if _, exists := debugData[key]; !exists {
// 			t.Errorf("expected key '%s' in /debug/vars output", key)
// 		}
// 	}

// 	// Verify StreamStatus is 1 (connected)
// 	if status, ok := debugData["stream_worker_connected"].(float64); !ok || status != 1 {
// 		t.Errorf("expected stream_worker_connected = 1, got %v", debugData["stream_worker_connected"])
// 	}

// 	// Verify stream_events_ingested_total has entries
// 	if eventsMap, ok := debugData["stream_events_ingested_total"].(map[string]any); !ok || len(eventsMap) == 0 {
// 		t.Errorf("expected stream_events_ingested_total to have entries, got %v", debugData["stream_events_ingested_total"])
// 	}

// 	// Verify http_requests_total >= 1
// 	if reqs, ok := debugData["http_requests_total"].(float64); !ok || reqs <= 0 {
// 		t.Errorf("expected http_requests_total > 0, got %v", debugData["http_requests_total"])
// 	}

// 	// Verify http_requests_by_status has "200"
// 	if statusMap, ok := debugData["http_requests_by_status"].(map[string]any); !ok {
// 		t.Errorf("expected http_requests_by_status map, got %v", debugData["http_requests_by_status"])
// 	} else if count, ok := statusMap["200"].(float64); !ok || count <= 0 {
// 		t.Errorf("expected status 200 count > 0, got %v", statusMap["200"])
// 	}

// 	// Verify analysis_requests_by_dimension has "likes"
// 	if dimMap, ok := debugData["analysis_requests_by_dimension"].(map[string]any); !ok {
// 		t.Errorf("expected analysis_requests_by_dimension map, got %v", debugData["analysis_requests_by_dimension"])
// 	} else if count, ok := dimMap["likes"].(float64); !ok || count <= 0 {
// 		t.Errorf("expected likes dimension count > 0, got %v", dimMap["likes"])
// 	}
// }
