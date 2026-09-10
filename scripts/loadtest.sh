#!/usr/bin/env bash
#
# Run the load profile against a freshly seeded instance.
#
# The numbers in the README come from this script. It exists so they can be reproduced
# rather than believed: it brings up the dependencies, applies the migrations, seeds a
# tenant, mints a token, starts the server and runs k6 against it.
#
# Usage: scripts/loadtest.sh [port]
#
# RATE, DURATION, BURST_RATE and BURST_START override the profile. The load generator
# and the service share a machine here, so on a busy one the tail is measuring the
# machine rather than the service; lower RATE until the load average is below the core
# count before trusting a p95.
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

PORT="${1:-8099}"
BASE_URL="http://localhost:${PORT}"
JWT_SECRET="${JWT_SECRET:-loadtest-secret}"
SEED_EMAIL="owner@dev.keystone.local"
SEED_PASSWORD='DevPass123!'
BURST_EMAIL="education.admin@dev.keystone.dev"
SERVER_BIN="$(mktemp -t keystone-loadtest)"
SERVER_LOG="$(mktemp -t keystone-loadtest-log)"
SERVER_PID=""

cleanup() {
  [ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
  rm -f "$SERVER_BIN"
}
trap cleanup EXIT

command -v k6 >/dev/null || { echo "k6 is not installed: brew install k6" >&2; exit 1; }

echo "==> dependencies"
docker compose up -d postgres redis >/dev/null
until docker compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done

echo "==> schema and seed data"
scripts/run_migrations.sh >/dev/null
docker compose exec -T postgres psql -U postgres -d keystone_dev < scripts/seed/dev_seed.sql >/dev/null

echo "==> building"
go build -o "$SERVER_BIN" ./cmd/server

echo "==> starting the server on port ${PORT}"
ENVIRONMENT=development JWT_SECRET="$JWT_SECRET" PORT="$PORT" "$SERVER_BIN" >"$SERVER_LOG" 2>&1 &
SERVER_PID=$!

# A port already in use is the failure worth guarding: the run would otherwise measure
# whatever else is listening. Waiting on /health from this build is what distinguishes
# them, since another service answers it differently or not at all.
for _ in $(seq 1 30); do
  if curl -sf "${BASE_URL}/health" | grep -q '"status":"ok"'; then break; fi
  sleep 1
done

curl -sf "${BASE_URL}/health" | grep -q '"status":"ok"' || {
  echo "the server did not come up on ${PORT}; log follows" >&2
  cat "$SERVER_LOG" >&2
  exit 1
}

mint_token() {
  curl -sf -X POST "${BASE_URL}/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$1\",\"password\":\"${SEED_PASSWORD}\"}" |
    python3 -c 'import sys,json; print(json.load(sys.stdin)["access_token"])'
}

echo "==> minting tokens"
# Two tenants on two plans. The steady phase runs on the enterprise tenant, the burst on
# the pro tenant, whose smaller bucket is what makes the limiter observable.
TOKEN=$(mint_token "$SEED_EMAIL")
BURST_TOKEN=$(mint_token "$BURST_EMAIL")

[ -n "$TOKEN" ] && [ -n "$BURST_TOKEN" ] || { echo "could not obtain a token" >&2; exit 1; }

echo "==> running k6"
BASE_URL="$BASE_URL" TOKEN="$TOKEN" BURST_TOKEN="$BURST_TOKEN" \
  RATE="${RATE:-100}" DURATION="${DURATION:-60s}" BURST_RATE="${BURST_RATE:-300}" \
  BURST_START="${BURST_START:-65s}" \
  k6 run loadtest/tenant_read.js

echo
echo "==> metrics after the run"
curl -s "${BASE_URL}/metrics" | grep -E '^keystone_(http_request_duration_seconds_count|http_requests_total|cache_events_total|rate_limit)' || true
