// Package metrics exposes the numbers this service is judged on.
//
// The repository made several performance claims before this package existed, and none
// of them were measured. Two decision records said so in as many words: ADR-0002 noted
// that the cost of the claim query was assumed rather than measured, and ADR-0006 noted
// that a comment claiming a "98-99% hit rate" had never been checked. A claim nobody
// can verify is worth less than a smaller claim somebody can.
//
// # Cardinality
//
// Nothing here is labelled by tenant id, and that is deliberate.
//
// A Prometheus time series is created for every distinct combination of label values.
// Labelling by tenant means the series count grows with the customer count, multiplied
// by every other label on the same metric. At a thousand tenants, the RED metrics alone
// become hundreds of thousands of series, and the monitoring system becomes the most
// expensive part of the deployment. Worse, it fails quietly: the metric keeps working
// while the storage bill grows.
//
// So the labels here are bounded by construction — route templates, not paths; plan
// names, not tenant ids; a small fixed set of outcomes. Per-tenant questions ("which
// customer is slow?") belong in traces and logs, which are sampled and indexed for
// exactly that. This is the standard trade-off, and it is written down because the
// original plan for this phase said to label by tenant id.
package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Registry holds every collector this service publishes.
//
// A dedicated registry is used rather than the default global one, so that tests can
// build an isolated instance and assert on it without leaking state between them.
type Registry struct {
	registry *prometheus.Registry

	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
	httpInFlight prometheus.Gauge

	jobsClaimed  *prometheus.CounterVec
	jobsFinished *prometheus.CounterVec
	jobDuration  *prometheus.HistogramVec
	jobsInFlight prometheus.Gauge
	queueDepth   *prometheus.GaugeVec

	cacheEvents *prometheus.CounterVec
	rateLimit   *prometheus.CounterVec
}

// httpBuckets covers the range this service actually operates in.
//
// The default Prometheus buckets top out at 10s, which wastes resolution on a service
// whose target is a P95 under 200ms: almost every observation lands in the first two
// buckets and the quantile estimate between them is a guess. These are tightened around
// the target instead, with enough headroom above it to see a regression.
var httpBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.5, 1, 2.5, 5,
}

// jobBuckets span a different scale. Provisioning creates a schema and runs migrations,
// so it is measured in seconds, not milliseconds.
var jobBuckets = []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120}

// New builds a registry with every collector registered.
func New() *Registry {
	r := &Registry{registry: prometheus.NewRegistry()}

	r.httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "keystone_http_requests_total",
		Help: "HTTP requests by route, method and status class.",
	}, []string{"method", "route", "status"})

	r.httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "keystone_http_request_duration_seconds",
		Help:    "HTTP request latency by route and method.",
		Buckets: httpBuckets,
	}, []string{"method", "route"})

	r.httpInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "keystone_http_requests_in_flight",
		Help: "HTTP requests currently being served.",
	})

	r.jobsClaimed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "keystone_worker_jobs_claimed_total",
		Help: "Jobs claimed by a worker, by kind.",
	}, []string{"kind"})

	r.jobsFinished = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "keystone_worker_jobs_finished_total",
		Help: "Jobs that reached an outcome, by kind and outcome.",
	}, []string{"kind", "outcome"})

	r.jobDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "keystone_worker_job_duration_seconds",
		Help:    "Time from claim to outcome, by kind.",
		Buckets: jobBuckets,
	}, []string{"kind"})

	r.jobsInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "keystone_worker_jobs_in_flight",
		Help: "Jobs currently running in this worker.",
	})

	// Queue depth is the number the on-call engineer actually looks at: it answers
	// whether the workers are keeping up, which no counter can.
	r.queueDepth = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "keystone_worker_queue_depth",
		Help: "Jobs waiting to be claimed, by status.",
	}, []string{"status"})

	r.cacheEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "keystone_cache_events_total",
		Help: "Cache outcomes by cache name and event.",
	}, []string{"cache", "event"})

	r.rateLimit = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "keystone_rate_limit_decisions_total",
		Help: "Rate limiter decisions, by plan and outcome.",
	}, []string{"plan", "outcome"})

	r.registry.MustRegister(
		r.httpRequests, r.httpDuration, r.httpInFlight,
		r.jobsClaimed, r.jobsFinished, r.jobDuration, r.jobsInFlight, r.queueDepth,
		r.cacheEvents, r.rateLimit,
	)

	return r
}

// Gatherer exposes the collectors for an exporter to scrape.
func (r *Registry) Gatherer() prometheus.Gatherer { return r.registry }

// --- HTTP ------------------------------------------------------------------

// HTTPStarted records a request entering the handler chain, and returns the function
// that records its outcome.
//
// The route must be the registered route template ("/api/v1/tenants/:id"), never the
// resolved path. A resolved path carries the id, and an id-per-series is the cardinality
// explosion this package is written to avoid.
func (r *Registry) HTTPStarted() func(method, route string, status int, elapsed time.Duration) {
	r.httpInFlight.Inc()

	return func(method, route string, status int, elapsed time.Duration) {
		r.httpInFlight.Dec()
		r.httpRequests.WithLabelValues(method, route, statusClass(status)).Inc()
		r.httpDuration.WithLabelValues(method, route).Observe(elapsed.Seconds())
	}
}

// statusClass reduces a status code to its class.
//
// Recording the exact code would multiply the series count for no operational gain: the
// alert that matters is "5xx is rising", not "422 is rising". The exact code is in the
// logs, which are not stored per-combination.
func statusClass(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	case status >= 200:
		return "2xx"
	default:
		return strconv.Itoa(status)
	}
}

// --- worker ----------------------------------------------------------------

// JobClaimed records a claim and returns the function that records the outcome.
func (r *Registry) JobClaimed(kind string) func(outcome string, elapsed time.Duration) {
	r.jobsClaimed.WithLabelValues(kind).Inc()
	r.jobsInFlight.Inc()

	return func(outcome string, elapsed time.Duration) {
		r.jobsInFlight.Dec()
		r.jobsFinished.WithLabelValues(kind, outcome).Inc()
		r.jobDuration.WithLabelValues(kind).Observe(elapsed.Seconds())
	}
}

// SetQueueDepth publishes how many jobs are waiting in a given status.
func (r *Registry) SetQueueDepth(status string, depth int) {
	r.queueDepth.WithLabelValues(status).Set(float64(depth))
}

// --- cache and rate limiting -----------------------------------------------

// CacheEvent records one cache outcome: "hit_l1", "hit_l2", "miss", "negative".
//
// This is what ADR-0006 was missing. Stats() already counted these in memory; until they
// were exported, the hit rate could be inspected in a debugger and nowhere else.
func (r *Registry) CacheEvent(cache, event string) {
	r.cacheEvents.WithLabelValues(cache, event).Inc()
}

// RateLimitDecision records an allow or deny, by plan.
//
// The plan is a bounded label — there are five of them — so it carries the useful part
// of the tenant question without the cardinality of a tenant id.
func (r *Registry) RateLimitDecision(plan string, allowed bool) {
	outcome := "allowed"
	if !allowed {
		outcome = "denied"
	}

	r.rateLimit.WithLabelValues(plan, outcome).Inc()
}
