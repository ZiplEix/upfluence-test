// Package server provides an HTTP API server wrapper with configurable options,
// lifecycle management, and built-in routing.
package server

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"

	"github.com/ZiplEix/upfluence-test/configuration"
	"github.com/ZiplEix/upfluence-test/service/aggregation"
)

var (
	ErrServerAlreadyRunning = errors.New("server already running")
)

// Option defines a functional configuration option for API.
type Option func(*API)

// API represents the HTTP API server instance, managing the underlying http.Server,
// request routing, associated services, configuration, and running state.
type API struct {
	config *configuration.Configuration
	server *http.Server
	mux    *http.ServeMux

	aggregation *aggregation.AggregationService

	running atomic.Bool
}

// WithConfig sets the configuration instance for the API.
func WithConfig(cfg *configuration.Configuration) Option {
	return func(a *API) {
		a.config = cfg
	}
}

// WithServer overrides the default http.Server instance.
func WithServer(s *http.Server) Option {
	return func(a *API) {
		a.server = s
	}
}

// WithAggregation injects an AggregationService dependency into the API.
func WithAggregation(svc *aggregation.AggregationService) Option {
	return func(a *API) {
		a.aggregation = svc
	}
}

// New initializes and returns a new API instance configured with default settings
// and any provided functional options.
func New(opts ...Option) *API {
	cfg := configuration.New()

	api := &API{
		config: cfg,
		server: &http.Server{
			Addr: cfg.Addr(),
		},
		mux: http.NewServeMux(),
	}

	for _, opt := range opts {
		opt(api)
	}

	if api.config != nil && api.server != nil && api.server.Addr == "" {
		api.server.Addr = api.config.Addr()
	}

	api.routes()

	api.server.Handler = loggingMiddleware(api.mux)

	return api
}

// Serve starts listening and serving incoming HTTP connections.
// Returns ErrServerAlreadyRunning if called while already running.
// Returns nil if the server was gracefully stopped.
func (a *API) Serve() error {
	if !a.running.CompareAndSwap(false, true) {
		return ErrServerAlreadyRunning
	}
	defer a.running.Store(false)

	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil // stoped by Shutdown() or Close()
	}

	return err
}

// Shutdown gracefully shuts down the server without interrupting active connections.
func (a *API) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}

// GetAddr returns the configured network address the server listens on.
// Returns an empty string if the API or underlying server is nil.
func (a *API) GetAddr() string {
	if a == nil || a.server == nil {
		return ""
	}

	return a.server.Addr
}

// Config returns the configuration instance associated with the API.
func (a *API) Config() *configuration.Configuration {
	if a == nil {
		return nil
	}

	return a.config
}
