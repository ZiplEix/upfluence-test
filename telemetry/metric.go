package telemetry

import "expvar"

var (
	// EventBus
	ActiveSubscribers = expvar.NewInt("eventbus_active_subscribers")
	DroppedEvents     = expvar.NewInt("eventbus_dropped_events_total")

	// Stream Worker
	StreamStatus        = expvar.NewInt("stream_worker_connected")
	StreamReconnections = expvar.NewInt("stream_worker_reconnections_total")
	StreamErrors        = expvar.NewInt("stream_worker_errors_total")
	EventsIngested      = expvar.NewMap("stream_events_ingested_total")

	// HTTP & Handlers
	HTTPRequestsTotal    = expvar.NewInt("http_requests_total")
	HTTPRequestsByStatus = expvar.NewMap("http_requests_by_status")
	AnalysisByDimension  = expvar.NewMap("analysis_requests_by_dimension")
)
