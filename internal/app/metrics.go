package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/canakyuz/keystone/pkg/metrics"
)

// registerMetrics exposes the Prometheus endpoint.
//
// It is registered outside the /api/v1 group and carries no authentication, which is the
// convention scrapers expect. That is safe only because the endpoint is not supposed to
// be reachable from the internet: in a real deployment it is bound to an internal
// listener or blocked at the edge. The metric names and labels here are deliberately
// free of tenant identifiers, so a leak would expose service-level rates and latencies
// rather than anything about a customer.
//
// It is also exempt from rate limiting. A scraper hitting its own limit would produce
// gaps in the very dashboards used to diagnose why traffic is being limited.
func registerMetrics(app *fiber.App, reg *metrics.Registry) {
	if reg == nil {
		return
	}

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.HandlerFor(
		reg.Gatherer(),
		promhttp.HandlerOpts{},
	)))
}
