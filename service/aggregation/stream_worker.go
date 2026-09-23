package aggregation

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ZiplEix/upfluence-test/eventbus"
	"github.com/ZiplEix/upfluence-test/telemetry"
)

// StreamWorker handles consuming an external HTTP event stream and publishing items to an event bus.
type StreamWorker struct {
	streamURL  string
	httpClient *http.Client
	bus        *eventbus.EventBus[Item]
}

// NewStreamWorker initializes and returns a new StreamWorker configured with the target stream URL,
// an infinite-timeout HTTP client, and the destination event bus.
func NewStreamWorker(streamURL string, bus *eventbus.EventBus[Item]) *StreamWorker {
	return &StreamWorker{
		streamURL: streamURL,
		httpClient: &http.Client{
			Timeout: 0,
		},
		bus: bus,
	}
}

// Start lunch a listen loop and automatic reconnexion in background
func (w *StreamWorker) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := w.consumeStream(ctx); err != nil && !errors.Is(err, context.Canceled) {
					slog.Error("stream connection failed, reconnecting in 2s", slog.String("error", err.Error()))
					telemetry.StreamReconnections.Add(1)
					time.Sleep(2 * time.Second)
				}
			}

		}
	}()
}

func (w *StreamWorker) consumeStream(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.streamURL, nil)
	if err != nil {
		telemetry.StreamErrors.Add(1)
		return err
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		telemetry.StreamErrors.Add(1)
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		telemetry.StreamErrors.Add(1)
		return errors.New("bad upstream status")
	}

	telemetry.StreamStatus.Set(1)
	defer telemetry.StreamStatus.Set(0)

	reader := bufio.NewReader(resp.Body)
	const prefix = "data: "

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			telemetry.StreamErrors.Add(1)
			return err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 || !bytes.HasPrefix(line, []byte(prefix)) {
			continue
		}

		payload := line[len(prefix):]
		var event StreamEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			telemetry.StreamErrors.Add(1)
			continue
		}

		item, ok := event.AsItem()
		if !ok {
			telemetry.EventsIngested.Add("ignored", 1)
			continue
		}

		telemetry.EventsIngested.Add(item.Platform(), 1)
		w.bus.Publish(item)
	}
}
