// Package tracing wires distributed tracing for the API and the worker.
//
// The reason this exists in a repository that already has metrics: metrics answer "how
// often, how fast, how many", and cannot answer "what happened to this one request".
// Provisioning is asynchronous, so the interesting failures are the ones that span two
// processes and several minutes, and those are exactly the ones a counter cannot
// describe.
//
// Tracing is off unless an endpoint is configured. Off means a no-op tracer, not a
// branch at every call site: the instrumentation stays in the code path either way, so
// the traced and untraced builds cannot drift.
package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Config describes where traces go and how much of them.
type Config struct {
	// Endpoint is the OTLP/HTTP collector address, "localhost:4318" for example.
	// Empty disables tracing.
	Endpoint string

	// ServiceName distinguishes the API from the worker in the collector. They are
	// separate processes and their spans must not be attributed to one service.
	ServiceName string

	// Environment tags every span, so a staging trace cannot be mistaken for production.
	Environment string

	// SampleRatio is the head sampling ratio, 0 to 1.
	//
	// Head sampling is used rather than tail sampling because the decision has to be made
	// before the first span is created, and it has to be the same decision in both
	// processes: a request sampled by the API whose worker half is dropped produces a
	// trace that is worse than none, since it looks like the work never happened.
	SampleRatio float64
}

// Init installs the global tracer provider and returns its shutdown function.
//
// With no endpoint it installs the propagator and nothing else. The propagator still
// matters: it is what lets a caller's trace context flow through this service even when
// this service is not recording, so an untraced hop does not break someone else's trace.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	// The propagator is global and format-specific. W3C TraceContext is the one every
	// modern collector and client library speaks; setting it explicitly rather than
	// relying on the default keeps the wire format from changing under a dependency
	// upgrade.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if cfg.Endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create the trace exporter: %w", err)
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		attribute.String("deployment.environment", cfg.Environment),
	))
	if err != nil {
		return nil, fmt.Errorf("could not build the trace resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		// Batched rather than synchronous: an exporter that blocks the request path turns
		// a monitoring outage into a service outage.
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(5*time.Second)),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(cfg.SampleRatio),
		)),
	)

	otel.SetTracerProvider(provider)

	return provider.Shutdown, nil
}

// Tracer returns a named tracer.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// storedFormat is the propagator used for the database hop.
//
// It is a package-level value rather than the global propagator on purpose. The column
// these functions read and write holds a W3C traceparent — migration 034 says so, and
// rows already in the table are in that format. Using the global propagator would mean
// the stored format changes if someone reconfigures propagation for the HTTP hop, and
// it would silently produce nothing at all if the global one had not been installed yet,
// because the OpenTelemetry default is a no-op. Both failures are invisible: the job
// still runs, the link is just never made.
var storedFormat = propagation.TraceContext{}

// Carrier is a map that carries trace context across a process boundary.
//
// The HTTP hop has headers for this. The provisioning hop does not: the producer writes
// a database row and the consumer reads it back, possibly on another machine, minutes
// or hours later. So the context travels as a string on that row.
type Carrier map[string]string

// Get implements propagation.TextMapCarrier.
func (c Carrier) Get(key string) string { return c[key] }

// Set implements propagation.TextMapCarrier.
func (c Carrier) Set(key, value string) { c[key] = value }

// Keys implements propagation.TextMapCarrier.
func (c Carrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}

	return keys
}

// Marshal serialises the trace context of ctx into a traceparent string.
//
// It returns the empty string when there is nothing to carry, so a caller can store the
// result without checking whether tracing is on.
func Marshal(ctx context.Context) string {
	carrier := Carrier{}
	storedFormat.Inject(ctx, carrier)

	return carrier["traceparent"]
}

// Unmarshal rebuilds a trace context from a traceparent string.
//
// An empty or malformed value yields a context with no span, which is the correct
// outcome: a job whose trace context was never recorded still has to run.
func Unmarshal(ctx context.Context, traceparent string) context.Context {
	if traceparent == "" {
		return ctx
	}

	return storedFormat.Extract(ctx, Carrier{"traceparent": traceparent})
}

// LinkFrom turns a stored traceparent into a span link.
//
// WHY a link and not a parent: the request that created the job has already finished.
// Making the worker's span a child of it would keep one trace open for as long as the
// job takes to succeed, which for a job that retries is measured in hours, and every
// retry would appear as another child of a request that ended long ago. Worse, a trace
// only completes when all its spans do, so a job that never succeeds leaves a trace that
// never closes.
//
// A link says "this work was caused by that request" without pretending the two are one
// operation. It is what the OpenTelemetry conventions call for when producer and
// consumer are decoupled in time, which is the whole point of the queue.
func LinkFrom(traceparent string) []trace.Link {
	if traceparent == "" {
		return nil
	}

	sc := trace.SpanContextFromContext(Unmarshal(context.Background(), traceparent))
	if !sc.IsValid() {
		return nil
	}

	return []trace.Link{{SpanContext: sc}}
}

// TraceID returns the current trace id, or the empty string.
//
// It is used to put the trace id into the log line, which is what makes a log search and
// a trace search reach the same incident.
func TraceID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}

	return sc.TraceID().String()
}
