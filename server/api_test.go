package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ZiplEix/upfluence-test/configuration"
	"github.com/ZiplEix/upfluence-test/eventbus"
	"github.com/ZiplEix/upfluence-test/service/aggregation"
)

func TestNew_Defaults(t *testing.T) {
	api := New()
	if api == nil {
		t.Fatal("expected non-nil API")
	}
	if api.server == nil {
		t.Fatal("expected non-nil underlying server")
	}
	if api.server.Addr != ":8080" {
		t.Errorf("expected default addr :8080, got %s", api.server.Addr)
	}
	if api.mux == nil {
		t.Error("expected non-nil mux")
	}
	if api.server.Handler == nil {
		t.Error("expected non-nil server handler")
	}
	if addr := api.GetAddr(); addr != ":8080" {
		t.Errorf("expected GetAddr :8080, got %s", addr)
	}
	if api.Config() == nil {
		t.Fatal("expected non-nil config")
	}
	if api.Config().Port != "8080" {
		t.Errorf("expected default Port '8080', got %s", api.Config().Port)
	}
	if api.Config().StreamURL != "https://stream.upfluence.co/stream" {
		t.Errorf("expected default StreamURL 'https://stream.upfluence.co/stream', got %s", api.Config().StreamURL)
	}
	if api.Config().EventBusBufferSize != 100 {
		t.Errorf("expected default EventBusBufferSize 100, got %d", api.Config().EventBusBufferSize)
	}
}

func TestNew_WithOptions(t *testing.T) {
	customServer := &http.Server{Addr: ":9999"}
	customConfig := &configuration.Configuration{
		Port:               "9999",
		StreamURL:          "https://custom.stream/events",
		EventBusBufferSize: 200,
	}
	bus := eventbus.New[aggregation.Item](10)
	svc := aggregation.New(bus)

	api := New(
		WithConfig(customConfig),
		WithServer(customServer),
		WithAggregation(svc),
	)

	if api.config != customConfig {
		t.Errorf("expected config %v, got %v", customConfig, api.config)
	}
	if api.Config() != customConfig {
		t.Errorf("expected Config() %v, got %v", customConfig, api.Config())
	}
	if api.server != customServer {
		t.Errorf("expected server %v, got %v", customServer, api.server)
	}
	if api.aggregation != svc {
		t.Errorf("expected aggregation %v, got %v", svc, api.aggregation)
	}
	if api.GetAddr() != ":9999" {
		t.Errorf("expected addr :9999, got %s", api.GetAddr())
	}
}

func TestServe_And_Shutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	bus := eventbus.New[aggregation.Item](10)
	svc := aggregation.New(bus)

	customServer := &http.Server{
		Addr: addr,
	}

	api := New(
		WithServer(customServer),
		WithAggregation(svc),
	)

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- api.Serve()
	}()

	// Wait until server is listening
	for {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Verify that calling Serve() again while running returns ErrServerAlreadyRunning
	errAlreadyRunning := api.Serve()
	if !errors.Is(errAlreadyRunning, ErrServerAlreadyRunning) {
		t.Errorf("expected ErrServerAlreadyRunning, got %v", errAlreadyRunning)
	}

	// Shutdown the server gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := api.Shutdown(ctx); err != nil {
		t.Fatalf("failed to shutdown gracefully: %v", err)
	}

	// Serve() should return nil (because ErrServerClosed is handled)
	select {
	case err := <-serveErrCh:
		if err != nil {
			t.Errorf("expected nil error on shutdown, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Serve to terminate after shutdown")
	}
}

func TestServe_ListenError(t *testing.T) {
	// Occupy a port first
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	occupiedAddr := ln.Addr().String()

	api := New(
		WithServer(&http.Server{Addr: occupiedAddr}),
	)

	err = api.Serve()
	if err == nil {
		t.Fatal("expected error listening on occupied address, got nil")
	}
}

func TestAPI_Config_Nil(t *testing.T) {
	var api *API
	if api.Config() != nil {
		t.Error("expected nil from nil API Config()")
	}
}
