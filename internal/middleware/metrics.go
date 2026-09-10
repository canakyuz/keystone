package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/canakyuz/keystone/pkg/metrics"
)

// normaliseRoute reduces a matched route to one label per endpoint.
//
// c.Route() answers differently depending on how far the request got: a request rejected
// by group middleware reports the group's path ("/api/v1/users"), while one that reached
// the handler reports the handler's ("/api/v1/users/"). Left alone that splits a single
// endpoint across two series, so the rejected requests and the served ones cannot be
// compared, which is the comparison a rate limit dashboard is for.
//
// The root path is left as "/" rather than collapsed to the empty string.
func normaliseRoute(path string) string {
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		return strings.Clone(strings.TrimSuffix(path, "/"))
	}

	return strings.Clone(path)
}

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

		// Cloned, not referenced.
		//
		// c.Method() returns a string that points into fasthttp's request buffer, which is
		// pooled and reused between requests. Holding it past the handler and storing it as
		// a metric label meant the label could be rewritten by a later request: a real run
		// produced the label "GETT", a "GET" whose backing bytes had been overwritten by the
		// next request's method. A corrupt label is worse than a missing one, because it
		// silently creates a series nothing will ever match again.
		method := strings.Clone(c.Method())

		err := c.Next()

		// The route template, not the resolved path. c.Path() would carry the id, and a
		// series per id is the cardinality problem pkg/metrics exists to avoid.
		//
		// A request that matched no route has an empty template. Those are reported under
		// a single "unmatched" label rather than their paths, because an unmatched path is
		// attacker-controlled: recording it verbatim would let anyone create unbounded
		// series by requesting random URLs.
		// The route template comes from the app's route table rather than the request, so it
		// is not pooled. It is cloned anyway: the cost is one small allocation per request,
		// and the alternative is depending on an implementation detail of the router.
		route := "unmatched"
		if r := c.Route(); r != nil && r.Path != "" {
			route = normaliseRoute(r.Path)
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

		finish(method, route, status, time.Since(start))

		return err
	}
}
