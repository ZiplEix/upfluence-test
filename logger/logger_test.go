package logger

import (
	"context"
	"log/slog"
	"sync"
	"testing"
)

func TestWithContext(t *testing.T) {
	ctx := WithContext(context.Background())
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	attrs := AttrsFromContext(ctx)
	if attrs == nil {
		t.Fatal("expected non-nil attrs slice from initialized context")
	}
	if len(attrs) != 0 {
		t.Fatalf("expected 0 attrs, got %d", len(attrs))
	}
}

func TestAdd_And_AttrsFromContext(t *testing.T) {
	ctx := WithContext(context.Background())

	Add(ctx, slog.String("env", "test"), slog.Int("count", 10))

	attrs := AttrsFromContext(ctx)
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}

	if attrs[0].Key != "env" || attrs[0].Value.String() != "test" {
		t.Errorf("unexpected attr[0]: %v", attrs[0])
	}
	if attrs[1].Key != "count" || attrs[1].Value.Int64() != 10 {
		t.Errorf("unexpected attr[1]: %v", attrs[1])
	}

	// Verify defensive copy: modifying returned slice should not alter the context
	attrs[0] = slog.String("env", "tampered")
	reloaded := AttrsFromContext(ctx)
	if reloaded[0].Value.String() != "test" {
		t.Errorf("expected defensive copy to prevent mutation, but got: %v", reloaded[0].Value.String())
	}
}

func TestAdd_NoStore(t *testing.T) {
	// Should gracefully do nothing and not panic when called with background context
	Add(context.Background(), slog.String("k", "v"))
}

func TestAttrsFromContext_NoStore(t *testing.T) {
	// Should return nil when context does not have a store initialized
	if attrs := AttrsFromContext(context.Background()); attrs != nil {
		t.Errorf("expected nil attrs from background context, got %v", attrs)
	}
}

func TestFromContext(t *testing.T) {
	// Uninitialized context returns default logger
	defaultLogger := slog.Default()
	l1 := FromContext(context.Background())
	if l1 != defaultLogger {
		t.Errorf("expected default logger for uninitialized context")
	}

	// Initialized context with empty attrs returns default logger
	ctx := WithContext(context.Background())
	l2 := FromContext(ctx)
	if l2 != defaultLogger {
		t.Errorf("expected default logger for empty attrs context")
	}

	// Context with attributes returns enriched logger
	Add(ctx, slog.String("service", "aggregation"), slog.Int("port", 8080))
	l3 := FromContext(ctx)
	if l3 == nil {
		t.Fatal("expected non-nil enriched logger")
	}
	if l3 == defaultLogger {
		t.Errorf("expected new enriched logger instance, got default logger")
	}
}

func TestLogger_Concurrency(t *testing.T) {
	ctx := WithContext(context.Background())
	var wg sync.WaitGroup

	const numGoroutines = 20
	const iterations = 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				Add(ctx, slog.Int("goroutine", id), slog.Int("iter", j))
				_ = AttrsFromContext(ctx)
				_ = FromContext(ctx)
			}
		}(i)
	}

	wg.Wait()

	attrs := AttrsFromContext(ctx)
	expectedCount := numGoroutines * iterations * 2
	if len(attrs) != expectedCount {
		t.Errorf("expected %d attrs, got %d", expectedCount, len(attrs))
	}
}
