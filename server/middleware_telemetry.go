package server

import (
	"net/http"
	"strconv"

	"github.com/ZiplEix/upfluence-test/telemetry"
)

type telemetryResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *telemetryResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func telemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		telemetry.HTTPRequestsTotal.Add(1)

		wrapped := &telemetryResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		telemetry.HTTPRequestsByStatus.Add(strconv.Itoa(wrapped.statusCode), 1)
	})
}
