package server

// func (a *API) registerTelemetryRoutes() {
// 	a.mux.Handle("GET /debug/vars", expvar.Handler())
// }

// func (a *API) registerHealthRoutes() {
// 	a.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusOK)
// 		_, _ = w.Write([]byte(`{"status":"ok"}`))
// 	})
// }

func (a *API) routes() {
	a.mux.HandleFunc("/analysis", a.handlerAggregation())

	// a.registerTelemetryRoutes()
	// a.registerHealthRoutes()
}
