package logger

import (
	"context"
	"log/slog"
	"sync"
)

type contextKey struct{}

// contextStore thread-safely stores attributes added during the request
type contextStore struct {
	mu    sync.Mutex
	attrs []slog.Attr
}

// WithContext initializes the attribute container in the context
func WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, &contextStore{
		attrs: make([]slog.Attr, 0, 4),
	})
}

// Add appends one or more attributes to the current request's context
func Add(ctx context.Context, attrs ...slog.Attr) {
	store, ok := ctx.Value(contextKey{}).(*contextStore)
	if !ok || store == nil {
		return
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.attrs = append(store.attrs, attrs...)
}

// AttrsFromContext extracts all accumulated attributes for the final log
func AttrsFromContext(ctx context.Context) []slog.Attr {
	store, ok := ctx.Value(contextKey{}).(*contextStore)
	if !ok || store == nil {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()

	// Retourne une copie défensive
	copied := make([]slog.Attr, len(store.attrs))
	copy(copied, store.attrs)
	return copied
}

// FromContext returns a *slog.Logger instance enriched with current attributes
func FromContext(ctx context.Context) *slog.Logger {
	attrs := AttrsFromContext(ctx)
	if len(attrs) == 0 {
		return slog.Default()
	}

	args := make([]any, len(attrs))
	for i, a := range attrs {
		args[i] = a
	}
	return slog.Default().With(args...)
}
