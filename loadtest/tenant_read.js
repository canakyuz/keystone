// Load profile for the tenant-scoped read path.
//
// This is the path the repository makes claims about. Every request on it goes through
// JWT verification, tenant resolution through the two-tier cache, the rate limiter, and
// then a query executed on a connection pinned to the tenant's schema. It is the hot
// path, so it is the one worth measuring.
//
// The probes are measured separately in the same run. /health touches nothing and
// /ready touches the database and Redis, so the gap between the two is the cost of the
// dependency checks rather than of the framework.
//
// Run it through scripts/loadtest.sh, which brings up the dependencies, seeds a tenant
// and mints a token first.
import http from 'k6/http';
import { check } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const BASE = __ENV.BASE_URL || 'http://localhost:8099';
const TOKEN = __ENV.TOKEN;

// A token for a tenant on a smaller plan, used only by the burst scenario. The steady
// scenario runs on an enterprise tenant whose bucket holds 6000 tokens, which is large
// enough to absorb any burst this machine can generate; measuring the limiter against
// it would report that the limiter does nothing. A pro tenant holds 1200, so the burst
// exceeds it and the rejection is real rather than arranged.
const BURST_TOKEN = __ENV.BURST_TOKEN || TOKEN;

// Overridable so the same profile can be run lighter on a busy machine. The defaults
// are the enterprise ceiling; the README records which values produced its numbers.
const RATE = Number(__ENV.RATE || 100);
const DURATION = __ENV.DURATION || '60s';
const BURST_RATE = Number(__ENV.BURST_RATE || 300);

// Separate trends per path. A single aggregate would average the database-backed read
// together with the liveness probe and report a number that describes neither.
const tenantReadTrend = new Trend('path_tenant_read', true);
const livenessTrend = new Trend('path_liveness', true);
const readinessTrend = new Trend('path_readiness', true);

// The share of burst requests the limiter turned away. Asserted below: a burst that is
// served in full means the quota is not being enforced, and "200 or 429" alone cannot
// tell the two apart because it passes either way.
const burstRejected = new Rate('burst_rejected');

export const options = {
  scenarios: {
    // A fixed arrival rate at the enterprise ceiling, which is 6000 requests a minute,
    // or 100 a second.
    //
    // An open ramp was tried first and measured the wrong thing: it drove roughly
    // 19,000 requests a second, the limiter rejected 99.94% of them, and the resulting
    // latency described how fast the service says no. Measuring above the quota the
    // product enforces produces a number that is real but answers no question anybody
    // has. The limiter is exercised deliberately in its own scenario below.
    tenant_read: {
      executor: 'constant-arrival-rate',
      exec: 'tenantRead',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
    // A short burst above the pro quota, started after the steady phase, to show the
    // limiter engaging rather than asserting that it would.
    //
    // 300 a second for 10 seconds is 3000 requests against a bucket of 1200 that refills
    // at 20 a second, so roughly half should be turned away. The token bucket allows the
    // first 1200 through on purpose: bursting up to capacity is the behaviour ADR-0007
    // chose over a fixed window.
    burst: {
      executor: 'constant-arrival-rate',
      exec: 'burst',
      rate: BURST_RATE,
      timeUnit: '1s',
      duration: '10s',
      startTime: __ENV.BURST_START || '65s',
      preAllocatedVUs: 50,
      maxVUs: 150,
    },

    // The probes run at a constant low rate throughout, the way a real orchestrator
    // polls them, so their numbers are not distorted by the load.
    probes: {
      executor: 'constant-arrival-rate',
      exec: 'probes',
      rate: 5,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 5,
    },
  },
  thresholds: {
    // The limiter must reject the burst, and must not reject the in-quota traffic. A
    // rate limit that is never exercised in a load test is an untested rate limit.
    'burst_rejected': ['rate>0.3'],
    // Asserted, not just reported. A load test whose output nobody reads is a load test
    // that silently stops being true.
    'path_tenant_read': ['p(95)<200'],
    // Scoped to the steady traffic. The burst scenario is expected to be rejected, so a
    // global failure rate would assert against the limiter doing its job.
    'http_req_failed{scenario:tenant_read}': ['rate<0.01'],
    'http_req_failed{scenario:probes}': ['rate<0.01'],
  },
};

export function tenantRead() {
  const res = http.get(`${BASE}/api/v1/users`, {
    headers: { Authorization: `Bearer ${TOKEN}` },
    tags: { path: 'tenant_read' },
  });

  tenantReadTrend.add(res.timings.duration);
  check(res, { 'tenant read is 200': (r) => r.status === 200 });
}

export function burst() {
  const res = http.get(`${BASE}/api/v1/users`, {
    headers: { Authorization: `Bearer ${BURST_TOKEN}` },
    tags: { path: 'burst' },
  });

  // Both outcomes are correct per request. What would not be correct is every request
  // passing, which is what the burst_rejected threshold asserts against.
  burstRejected.add(res.status === 429);
  check(res, { 'burst is 200 or 429': (r) => r.status === 200 || r.status === 429 });
}

export function probes() {
  const health = http.get(`${BASE}/health`, { tags: { path: 'liveness' } });
  livenessTrend.add(health.timings.duration);
  check(health, { 'liveness is 200': (r) => r.status === 200 });

  const ready = http.get(`${BASE}/ready`, { tags: { path: 'readiness' } });
  readinessTrend.add(ready.timings.duration);
  check(ready, { 'readiness is 200': (r) => r.status === 200 });
}
