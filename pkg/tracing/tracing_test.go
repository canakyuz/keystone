package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// withRecordingTracer installs a provider that always samples, so the tests observe real
// span contexts rather than the no-op ones a disabled setup produces.
func withRecordingTracer(t *testing.T) {
	t.Helper()

	_, err := Init(context.Background(), Config{})
	require.NoError(t, err)

	previous := otel.GetTracerProvider()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(provider)

	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		_ = provider.Shutdown(context.Background())
	})
}

// TestMarshalUnmarshal_RoundTripsTheTraceContext verifies the context survives the trip
// through a database column.
//
// This is the hop that has no headers: the API writes a row and the worker reads it back
// in another process. If the string does not round-trip, the two halves of a provisioning
// end up in unrelated traces and the feature silently does nothing.
func TestMarshalUnmarshal_RoundTripsTheTraceContext(t *testing.T) {
	withRecordingTracer(t)

	ctx, span := Tracer("test").Start(context.Background(), "producer")
	defer span.End()

	traceparent := Marshal(ctx)
	require.NotEmpty(t, traceparent, "nothing was captured to carry")

	restored := Unmarshal(context.Background(), traceparent)

	assert.Equal(t, TraceID(ctx), TraceID(restored),
		"the trace id did not survive the round trip")
}

// TestLinkFrom_ProducesALinkNotAParent verifies the stored context becomes a link.
//
// The distinction matters operationally. A parent would keep the request's trace open
// until the job finally succeeds, which for a retried job is hours, and every retry
// would hang off a request that ended long ago.
func TestLinkFrom_ProducesALinkNotAParent(t *testing.T) {
	withRecordingTracer(t)

	ctx, span := Tracer("test").Start(context.Background(), "producer")
	span.End()

	links := LinkFrom(Marshal(ctx))
	require.Len(t, links, 1)

	assert.Equal(t, TraceID(ctx), links[0].SpanContext.TraceID().String())

	// The consumer's own span belongs to a different trace: it is caused by the request,
	// not part of it.
	consumerCtx, consumerSpan := Tracer("test").Start(context.Background(), "consumer")
	defer consumerSpan.End()

	assert.NotEqual(t, TraceID(ctx), TraceID(consumerCtx),
		"the consumer joined the producer's trace instead of linking to it")
}

// TestUnmarshal_ToleratesMissingAndMalformedInput verifies a job with no usable trace
// context still runs.
//
// Operations created before tracing existed carry nothing, and a malformed value is
// what a truncated column or a partial write looks like. Neither is a reason to fail a
// provisioning.
func TestUnmarshal_ToleratesMissingAndMalformedInput(t *testing.T) {
	withRecordingTracer(t)

	for _, input := range []string{"", "not-a-traceparent", "00-invalid-00"} {
		restored := Unmarshal(context.Background(), input)

		assert.Empty(t, TraceID(restored), "input %q produced a trace id", input)
		assert.Nil(t, LinkFrom(input), "input %q produced a link", input)
	}
}

// TestMarshal_DoesNotDependOnGlobalSetup verifies the stored format works before Init.
//
// These functions read and write a column whose contents are already W3C traceparents.
// Routing them through the global propagator made them depend on initialisation order,
// and OpenTelemetry's default global propagator is a no-op, so the failure was silent:
// the job still ran, the link was simply never made. A test wrote a traceparent, read it
// back intact, and got no link out of it.
func TestMarshal_DoesNotDependOnGlobalSetup(t *testing.T) {
	const traceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	// Deliberately no Init, and the global propagator left as the no-op default.
	links := LinkFrom(traceparent)

	require.Len(t, links, 1, "the stored format fell back to the global propagator")
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736",
		links[0].SpanContext.TraceID().String())
}
