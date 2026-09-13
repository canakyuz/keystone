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
# TARGET_URL points the run at a service running somewhere else, which is the only way to
# get a tail figure that describes the service. With it set, nothing is built, started or
# seeded here: the host at the other end is expected to be running this build against a
# seeded database, and this machine only generates load and holds the results.
#
#   TARGET_URL=https://keystone.example.com scripts/loadtest.sh
#
# RATE, DURATION, BURST_RATE and BURST_START override the profile. Without TARGET_URL the
# generator and the service share a host; this script then measures how busy that host is
# and tells k6 whether the tail is worth asserting.
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

PORT="${1:-8099}"
TARGET_URL="${TARGET_URL:-}"
BASE_URL="${TARGET_URL:-http://localhost:${PORT}}"
REMOTE=0
[ -n "$TARGET_URL" ] && REMOTE=1
JWT_SECRET="${JWT_SECRET:-loadtest-secret}"
SEED_EMAIL="owner@dev.local"
SEED_PASSWORD='DevPass123!'
BURST_EMAIL="edu.admin@dev.local"
SERVER_BIN="$(mktemp -t keystone-loadtest)"
SERVER_LOG="$(mktemp -t keystone-loadtest-log)"
SERVER_PID=""

cleanup() {
  [ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
  rm -f "$SERVER_BIN"
}
trap cleanup EXIT

command -v k6 >/dev/null || { echo "k6 is not installed: brew install k6" >&2; exit 1; }

# How busy the host is, and whether it is also serving the requests.
#
# The tail is a property of the service only when the generator is not competing with it
# for the same cores. Rather than asking the reader to remember that, the run works it out
# and passes the answer to k6, which asserts the tail or merely reports it.
cores=$(getconf _NPROCESSORS_ONLN 2>/dev/null || sysctl -n hw.ncpu)
load=$(uptime | sed -E 's/.*load averages?: *([0-9.]+).*/\1/')
TAIL_TRUSTED=1
if [ "$REMOTE" -eq 0 ] && awk -v l="$load" -v c="$cores" 'BEGIN { exit !(l > c * 0.6) }'; then
  TAIL_TRUSTED=0
fi

echo "==> conditions"
echo "    target        ${BASE_URL}$([ "$REMOTE" -eq 1 ] && echo " (remote)" || echo " (same host as the generator)")"
echo "    cores         ${cores}"
echo "    load average  ${load}"
echo "    tail          $([ "$TAIL_TRUSTED" -eq 1 ] && echo "asserted" || echo "reported only, this host is too busy to measure it")"

if [ "$REMOTE" -eq 1 ]; then
  curl -sf "${BASE_URL}/health" | grep -q '"status":"ok"' || {
    echo "the target did not answer /health with status ok: ${BASE_URL}" >&2
    exit 1
  }
fi

if [ "$REMOTE" -eq 0 ]; then
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

fi

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
BASE_URL="$BASE_URL" TOKEN="$TOKEN" BURST_TOKEN="$BURST_TOKEN" TAIL_TRUSTED="$TAIL_TRUSTED" \
  RATE="${RATE:-100}" DURATION="${DURATION:-60s}" BURST_RATE="${BURST_RATE:-300}" \
  BURST_START="${BURST_START:-65s}" \
  k6 run loadtest/tenant_read.js

echo
echo "==> metrics after the run"
curl -s "${BASE_URL}/metrics" | grep -E '^keystone_(http_request_duration_seconds_count|http_requests_total|cache_events_total|rate_limit)' || true
