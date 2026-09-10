package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/metrics"
)

// serveMetrics starts the worker's metrics listener and returns the function that stops
// it.
//
// A failure to bind does not stop the worker. Losing observability is bad; refusing to
// drain the provisioning queue because a monitoring port was taken is worse, and it
// turns a monitoring problem into a product outage.
func serveMetrics(addr string, reg *metrics.Registry, log *logger.Logger) func() {
	if addr == "" || reg == nil {
		return func() {}
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg.Gatherer(), promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) && log != nil {
			log.WithFields(logger.Fields{
				"addr":  addr,
				"error": err.Error(),
			}).Warn("metrics listener stopped")
		}
	}()

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}
}
