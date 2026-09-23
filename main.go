package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZiplEix/upfluence-test/configuration"
	"github.com/ZiplEix/upfluence-test/eventbus"
	"github.com/ZiplEix/upfluence-test/server"
	"github.com/ZiplEix/upfluence-test/service/aggregation"
)

func main() {
	cfg := configuration.New()

	appCtx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	bus := eventbus.New[aggregation.Item](cfg.EventBusBufferSize)

	worker := aggregation.NewStreamWorker(cfg.StreamURL, bus)
	worker.Start(appCtx)

	aggregationService := aggregation.New(bus)

	s := &http.Server{
		Addr:         cfg.Addr(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0,
	}

	api := server.New(
		server.WithConfig(cfg),
		server.WithServer(s),
		server.WithAggregation(aggregationService),
	)

	go func() {
		log.Printf("Starting server on http://localhost%s\n", api.GetAddr())
		if err := api.Serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := api.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v\n", err)
	}
}
