package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/canakyuz/keystone/pkg/tracing"
)

// Tracing starts a server span for every request.
//
// It runs early, alongside the metrics middleware, so that a request rejected by the
// rate limiter or the authenticator still produces a span. A trace that only covers
// successful requests is missing precisely the requests anybody would open a tracing UI
// to look at.
//
// The incoming trace context is extracted from the request headers, so a caller that is
// already tracing keeps one trace across the hop rather than starting a second.
func Tracing(serviceName string) fiber.Handler {
	tracer := tracing.Tracer(serviceName)

	return func(c *fiber.Ctx) error {
		ctx := otel.GetTextMapPropagator().Extract(c.UserContext(), headerCarrier{c})

		// The span is named after the route template, not the path: a span name per id
		// makes the trace list unreadable and defeats grouping in every backend.
		name := c.Method() + " " + routeName(c)

		ctx, span := tracer.Start(ctx, name,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Method()),
				semconv.HTTPRoute(routeName(c)),
				semconv.URLPath(c.Path()),
			),
		)
		defer span.End()

		// Handlers read the tenant schema from the user context, so the span context has to
		// be written there rather than replacing it: dropping the tenant schema to make room
		// for a trace would trade a working request for an observable one.
		c.SetUserContext(ctx)

		// The trace id goes on the response so a report of "this request was slow" can be
		// turned into a trace lookup without guessing.
		if id := tracing.TraceID(ctx); id != "" {
			c.Set("X-Trace-ID", id)
		}

		err := c.Next()

		status := c.Response().StatusCode()
		span.SetAttributes(attribute.Int("http.response.status_code", status))

		// Only server errors mark the span as failed. A 404 or a 429 is the service working
		// as designed, and marking those failed makes the error rate in the tracing backend
		// disagree with the one in the metrics.
		switch {
		case err != nil:
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		case status >= 500:
			span.SetStatus(codes.Error, "server error")
		}

		return err
	}
}

// routeName returns the matched route template, or a constant for an unmatched request.
//
// An unmatched path is attacker-controlled, so it is never used as a span name: doing so
// would let anyone fill the trace backend's name index by requesting random URLs.
func routeName(c *fiber.Ctx) string {
	if r := c.Route(); r != nil && r.Path != "" {
		return r.Path
	}

	return "unmatched"
}

// headerCarrier adapts Fiber's request headers to the propagator's interface.
type headerCarrier struct{ c *fiber.Ctx }

func (h headerCarrier) Get(key string) string { return h.c.Get(key) }

func (h headerCarrier) Set(key, value string) { h.c.Set(key, value) }

func (h headerCarrier) Keys() []string {
	keys := make([]string, 0)
	h.c.Request().Header.VisitAll(func(key, _ []byte) {
		keys = append(keys, string(key))
	})

	return keys
}
