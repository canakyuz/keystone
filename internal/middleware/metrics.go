package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/canakyuz/keystone/pkg/metrics"
)

// Metrics records the RED signals — rate, errors, duration — for every request.
//
// It is registered before the other middleware so that a request rejected by the rate
// limiter or the authenticator is still counted. Placing it after them would make the
// dashboards disagree with reality in exactly the situation someone is looking at them:
// during an incident, when most requests are being rejected.
func Metrics(reg *metrics.Registry) fiber.Handler {
	if reg == nil {
		return func(c *fiber.Ctx) error { return c.Next() }
	}

	return func(c *fiber.Ctx) error {
		start := time.Now()
		finish := reg.HTTPStarted()

		err := c.Next()

		// The route template, not the resolved path. c.Path() would carry the id, and a
		// series per id is the cardinality problem pkg/metrics exists to avoid.
		//
		// A request that matched no route has an empty template. Those are reported under
		// a single "unmatched" label rather than their paths, because an unmatched path is
		// attacker-controlled: recording it verbatim would let anyone create unbounded
		// series by requesting random URLs.
		route := "unmatched"
		if r := c.Route(); r != nil && r.Path != "" {
			route = r.Path
		}

		status := c.Response().StatusCode()
		if err != nil {
			// The error handler has not run yet, so the response still carries the status
			// from before the failure. Fiber's own errors know better.
			if fe, ok := err.(*fiber.Error); ok {
				status = fe.Code
			} else {
				status = fiber.StatusInternalServerError
			}
		}

		finish(c.Method(), route, status, time.Since(start))

		return err
	}
}
