#!/usr/bin/env bash
# End-to-end check of the learning loop over HTTP, against a throwaway
# database and the deterministic mock tutor:
#   sign up → enrol (personalisation off) → roadmap → hand-written lesson steps one at a time
#   (Ask AI, 2 hints, give up, one similar question, cheat sheet saved to notes) → flashcards saved and
#   reviewed → checkpoint blocked until the steps are done → failing checkpoint (no unlock) → passing
#   checkpoint → summary + mistake flashcards → unlock → second user sees none of it → API restart →
#   log in again → progress, flashcards and notes still there.
# Requires: docker compose db running (infra/docker-compose.yml), Go, python3.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PORT="${E2E_PORT:-8090}"
API="http://localhost:$PORT"
ADMIN_URL="${E2E_ADMIN_URL:-postgres://fluentpath:fluentpath@localhost:5433/fluentpath?sslmode=disable}"
DB="fluentpath_e2e_$$"
DB_URL="$(printf %s "$ADMIN_URL" | sed "s#/fluentpath?#/$DB?#")"
BIN="$(mktemp -d)/server"
LOG="$(mktemp)"
PID=""

cleanup() {
  [ -n "$PID" ] && kill "$PID" 2>/dev/null || true
  psql "$ADMIN_URL" -qc "DROP DATABASE IF EXISTS $DB" >/dev/null 2>&1 || true
}
trap cleanup EXIT

start_api() {
  (cd "$ROOT/apps/api" && DATABASE_URL="$DB_URL" AI_PROVIDER=mock PORT="$PORT" AI_RATE_LIMIT_PER_MIN=10000 RATE_LIMIT_PER_MIN=10000 \
    COURSES_DIR="$ROOT/packages/courses" AUTH_RATE_LIMIT_PER_MIN=10000 exec "$BIN" >>"$LOG" 2>&1) &
  PID=$!
  for _ in $(seq 1 50); do
    curl -fs "$API/healthz" >/dev/null 2>&1 && return 0
    sleep 0.2
  done
  echo "API did not start; log:"; cat "$LOG"; exit 1
}

echo "▸ building API"
(cd "$ROOT/apps/api" && go build -o "$BIN" ./cmd/server)
echo "▸ creating database $DB"
psql "$ADMIN_URL" -qc "CREATE DATABASE $DB" >/dev/null

echo "▸ run 1: learn every ready lesson"
start_api
python3 "$ROOT/scripts/e2e_driver.py" learn "$API" "$ROOT/packages/courses/english"

echo "▸ restarting API"
kill "$PID"; wait "$PID" 2>/dev/null || true; PID=""
start_api
python3 "$ROOT/scripts/e2e_driver.py" verify "$API" "$ROOT/packages/courses/english"

echo "✓ e2e passed"
