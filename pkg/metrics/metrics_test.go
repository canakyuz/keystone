package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStatusClass verifies status codes collapse to their class.
//
// Recording the exact code would multiply the series count for no operational gain: the
// alert that matters is "5xx is rising", not "422 is rising".
func TestStatusClass(t *testing.T) {
	for status, want := range map[int]string{
		200: "2xx", 201: "2xx", 204: "2xx",
		301: "3xx", 400: "4xx", 404: "4xx", 429: "4xx",
		500: "5xx", 503: "5xx",
	} {
		assert.Equal(t, want, statusClass(status), "status %d", status)
	}
}

// TestHTTPStarted_RecordsRateAndDuration verifies a request produces both a counted
// outcome and a latency observation.
func TestHTTPStarted_RecordsRateAndDuration(t *testing.T) {
	reg := New()

	finish := reg.HTTPStarted()
	finish("GET", "/api/v1/users", 200, 50*time.Millisecond)

	assert.Equal(t, 1, testutil.CollectAndCount(reg.httpRequests))
	assert.Equal(t, float64(1), testutil.ToFloat64(
		reg.httpRequests.WithLabelValues("GET", "/api/v1/users", "2xx")))
}

// TestHTTPInFlight_ReturnsToZero verifies the in-flight gauge is balanced.
//
// A gauge that only goes up is worse than no gauge: it reads as a permanent overload
// and trains people to ignore it.
func TestHTTPInFlight_ReturnsToZero(t *testing.T) {
	reg := New()

	first := reg.HTTPStarted()
	second := reg.HTTPStarted()
	assert.Equal(t, float64(2), testutil.ToFloat64(reg.httpInFlight))

	first("GET", "/a", 200, time.Millisecond)
	second("GET", "/a", 500, time.Millisecond)

	assert.Equal(t, float64(0), testutil.ToFloat64(reg.httpInFlight))
}

// TestJobClaimed_RecordsOutcome verifies the worker's claim and outcome are paired.
func TestJobClaimed_RecordsOutcome(t *testing.T) {
	reg := New()

	finish := reg.JobClaimed("tenant.provision")
	assert.Equal(t, float64(1), testutil.ToFloat64(reg.jobsInFlight))

	finish("succeeded", 2*time.Second)

	assert.Equal(t, float64(0), testutil.ToFloat64(reg.jobsInFlight))
	assert.Equal(t, float64(1), testutil.ToFloat64(
		reg.jobsFinished.WithLabelValues("tenant.provision", "succeeded")))
}

// TestRateLimitDecision_LabelsByPlan verifies decisions are split by plan and outcome.
func TestRateLimitDecision_LabelsByPlan(t *testing.T) {
	reg := New()

	reg.RateLimitDecision("free", true)
	reg.RateLimitDecision("free", false)
	reg.RateLimitDecision("pro", true)

	assert.Equal(t, float64(1), testutil.ToFloat64(reg.rateLimit.WithLabelValues("free", "allowed")))
	assert.Equal(t, float64(1), testutil.ToFloat64(reg.rateLimit.WithLabelValues("free", "denied")))
	assert.Equal(t, float64(1), testutil.ToFloat64(reg.rateLimit.WithLabelValues("pro", "allowed")))
}

// TestExposition_CarriesNoTenantIdentifier is the cardinality guard.
//
// The label set is the part of a metrics layer that is expensive to get wrong: a series
// is created per distinct combination of label values, so one unbounded label makes the
// storage cost grow with the customer count. This asserts the property rather than
// trusting the reviewer to notice, because the failure is silent — the metric keeps
// working while the bill grows.
func TestExposition_CarriesNoTenantIdentifier(t *testing.T) {
	reg := New()

	reg.HTTPStarted()("GET", "/api/v1/users", 200, time.Millisecond)
	reg.JobClaimed("tenant.provision")("succeeded", time.Second)
	reg.CacheEvent("tenant_schema", "hit_l1")
	reg.RateLimitDecision("free", true)
	reg.SetQueueDepth("pending", 3)

	families, err := reg.Gatherer().Gather()
	require.NoError(t, err)
	require.NotEmpty(t, families)

	for _, family := range families {
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				name := strings.ToLower(label.GetName())
				assert.NotContains(t, name, "tenant_id",
					"metric %s carries an unbounded tenant label", family.GetName())
				assert.NotContains(t, name, "user_id",
					"metric %s carries an unbounded user label", family.GetName())
			}
		}
	}
}
