package server

func (a *API) routes() {
	a.mux.HandleFunc("/analysis", a.handlerAggregation())
}
