package aggregation

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ZiplEix/upfluence-test/eventbus"
)

func TestNewStreamWorker(t *testing.T) {
	bus := eventbus.New[Item](10)
	w := NewStreamWorker("http://example.com/stream", bus)
	if w == nil {
		t.Fatal("expected non-nil StreamWorker")
	}
	if w.streamURL != "http://example.com/stream" {
		t.Errorf("unexpected streamURL: %s", w.streamURL)
	}
	if w.bus != bus {
		t.Error("unexpected bus")
	}
	if w.httpClient == nil || w.httpClient.Timeout != 0 {
		t.Errorf("expected client with timeout 0, got %v", w.httpClient)
	}
}

func TestStreamWorker_ConsumeStream_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		_, _ = fmt.Fprint(w, "data: {\"tweet\":{\"id\":101,\"favorites\":50,\"retweets\":12,\"timestamp\":1000}}\n")
		_, _ = fmt.Fprint(w, "data: {\"youtube_video\":{\"id\":202,\"likes\":500,\"comments\":25,\"timestamp\":2000}}\n")
	}))
	defer ts.Close()

	bus := eventbus.New[Item](10)
	ch, unsub := bus.Subscribe()
	defer unsub()

	worker := NewStreamWorker(ts.URL, bus)
	ctx := context.Background()

	err := worker.consumeStream(ctx)
	if err != nil {
		t.Fatalf("expected nil error from consumeStream on EOF, got: %v", err)
	}

	// Verify both items were received
	item1 := <-ch
	if item1.Platform() != "tweet" {
		t.Errorf("expected item1 platform tweet, got %s", item1.Platform())
	}
	fav, _ := item1.Metric("favorites")
	if fav != 50 {
		t.Errorf("expected favorites 50, got %d", fav)
	}

	item2 := <-ch
	if item2.Platform() != "youtube_video" {
		t.Errorf("expected item2 platform youtube_video, got %s", item2.Platform())
	}
	likes, _ := item2.Metric("likes")
	if likes != 500 {
		t.Errorf("expected likes 500, got %d", likes)
	}
}

func TestStreamWorker_ConsumeStream_MalformedAndFiltered(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Empty lines
		_, _ = fmt.Fprint(w, "\n\n")
		// Missing prefix "data: "
		_, _ = fmt.Fprint(w, ":ping\n")
		_, _ = fmt.Fprint(w, "event: update\n")
		// Invalid JSON
		_, _ = fmt.Fprint(w, "data: {not-json}\n")
		// Empty event without items
		_, _ = fmt.Fprint(w, "data: {}\n")
		// Valid item
		_, _ = fmt.Fprint(w, "data: {\"pin\":{\"id\":5,\"likes\":88,\"comments\":3,\"timestamp\":3000}}\n")
	}))
	defer ts.Close()

	bus := eventbus.New[Item](10)
	ch, unsub := bus.Subscribe()
	defer unsub()

	worker := NewStreamWorker(ts.URL, bus)
	err := worker.consumeStream(context.Background())
	if err != nil {
		t.Fatalf("expected nil error on EOF, got: %v", err)
	}

	// Only 1 item should be received
	select {
	case item := <-ch:
		if item.Platform() != "pin" {
			t.Errorf("expected platform pin, got %s", item.Platform())
		}
	default:
		t.Fatal("expected 1 item, but got none")
	}

	// Ensure no extra items were published
	select {
	case extra := <-ch:
		t.Errorf("unexpected extra item: %v", extra)
	default:
		// Success
	}
}

func TestStreamWorker_ConsumeStream_BadStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	bus := eventbus.New[Item](10)
	worker := NewStreamWorker(ts.URL, bus)

	err := worker.consumeStream(context.Background())
	if err == nil {
		t.Fatal("expected error on 500 status, got nil")
	}
	if err.Error() != "bad upstream status" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestStreamWorker_ConsumeStream_ConnectionError(t *testing.T) {
	// Allocate a port and immediately close it so connection is refused immediately
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	bus := eventbus.New[Item](10)
	worker := NewStreamWorker("http://"+addr, bus)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = worker.consumeStream(ctx)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestStreamWorker_ConsumeStream_ContextCanceled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	}))
	defer ts.Close()

	bus := eventbus.New[Item](10)
	worker := NewStreamWorker(ts.URL, bus)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := worker.consumeStream(ctx)
	if err != nil {
		t.Errorf("expected nil error when context is canceled, got: %v", err)
	}
}

func TestStreamWorker_Start_Lifecycle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "data: {\"article\":{\"id\":10,\"likes\":123,\"comments\":4,\"timestamp\":5000}}\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	}))
	defer ts.Close()

	bus := eventbus.New[Item](10)
	ch, unsub := bus.Subscribe()
	defer unsub()

	worker := NewStreamWorker(ts.URL, bus)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	worker.Start(ctx)

	// Wait for item to arrive through worker.Start
	select {
	case item := <-ch:
		if item.Platform() != "article" {
			t.Errorf("expected platform article, got %s", item.Platform())
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for item from worker.Start")
	}

	// Cancel context to stop worker goroutine
	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestStreamWorker_Start_ReconnectOnError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	bus := eventbus.New[Item](10)
	worker := NewStreamWorker("http://"+addr, bus)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	worker.Start(ctx)
	<-ctx.Done()
}
