package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ZiplEix/upfluence-test/logger"
	"github.com/ZiplEix/upfluence-test/telemetry"
)

var (
	validMimensions = map[string]bool{
		"likes":     true,
		"comments":  true,
		"favorites": true,
		"retweets":  true,
	}
)

func (a *API) handlerAggregation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()

		durationParam := r.URL.Query().Get("duration")
		if durationParam != "" {
			d, err := time.ParseDuration(durationParam)
			if err != nil {
				errMsg := fmt.Sprintf("invalid duration parameter: %v", err)
				logger.Add(ctx, slog.String("error", errMsg))
				http.Error(w, errMsg, http.StatusBadRequest)
				return
			}

			if d < 0 {
				http.Error(w, "invalid duration parameter, duration must be > 0", http.StatusBadRequest)
				return
			}

			logger.Add(ctx, slog.String("requested_duration", durationParam))

			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, d)
			defer cancel()
		} else {
			logger.Add(ctx, slog.String("requested_duration", "infinite"))
		}

		dimensionParam := r.URL.Query().Get("dimension")
		if !validMimensions[dimensionParam] {
			http.Error(w, "invalid or missing dimension parameter (allowed: likes, comments, favorites, retweets)", http.StatusBadRequest)
			logger.Add(ctx, slog.String("error", fmt.Sprintf("invalid or missing dimension parameter, got '%s'", dimensionParam)))
			return
		}

		telemetry.AnalysisByDimension.Add(dimensionParam, 1)

		slog.Default().Log(ctx, slog.LevelInfo, "starting aggregation",
			slog.String("request_id", w.Header().Get("X-Request-ID")),
			slog.String("duration", durationParam),
		)

		report, err := a.aggregation.Analyze(ctx, dimensionParam)
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return
			}
			errMsg := fmt.Sprintf("aggregation failed: %v", err)
			logger.Add(ctx, slog.String("error", errMsg))
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report)
	}
}
