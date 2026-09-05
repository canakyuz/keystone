#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="$PROJECT_ROOT/migrations"

if [ ! -d "$MIGRATIONS_DIR" ]; then
  echo "Migrations dizini bulunamadi: $MIGRATIONS_DIR" >&2
  exit 1
fi

APP_ENV=${APP_ENV:-production}

DB_CMD=(docker compose exec -e PGOPTIONS="-c app.environment=${APP_ENV}" -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d keystone_dev)

run_psql() {
  "${DB_CMD[@]}" -c "$1"
}

ensure_schema_table() {
  run_psql "CREATE TABLE IF NOT EXISTS schema_migrations (id SERIAL PRIMARY KEY, filename TEXT UNIQUE NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW());"
}

schema_table_empty() {
  "${DB_CMD[@]}" -At -c "SELECT COUNT(*) = 0 FROM schema_migrations;"
}

table_exists() {
  local table_name="$1"
  local query="SELECT to_regclass('$table_name') IS NOT NULL;"
  "${DB_CMD[@]}" -At -c "$query"
}

bootstrap_migrations() {
  ensure_schema_table
  local inserted=0

  for file_path in "$MIGRATIONS_DIR"/*.up.sql; do
    [ -f "$file_path" ] || continue
    local filename
    filename=$(basename "$file_path")
    run_psql "INSERT INTO schema_migrations (filename) VALUES ('$filename') ON CONFLICT (filename) DO NOTHING;"
    inserted=$((inserted + 1))
  done

  echo "\033[32m[OK]\033[0m Bootstrap tamamlandi (kaydedilen dosya sayisi: $inserted)"
}

migration_applied() {
  local filename="$1"
  local query="SELECT 1 FROM schema_migrations WHERE filename = '$filename' LIMIT 1;"
  if "${DB_CMD[@]}" -At -c "$query" | grep -q 1; then
    return 0
  fi
  return 1
}

record_migration() {
  local filename="$1"
  local query="INSERT INTO schema_migrations (filename) VALUES ('$filename');"
  run_psql "$query"
}

apply_migration() {
  local file_path="$1"
  cat "$file_path" | "${DB_CMD[@]}"
}

main() {
  if [ "${BOOTSTRAP_MIGRATIONS:-0}" = "1" ]; then
    bootstrap_migrations
    return 0
  fi

  ensure_schema_table

  if [ "$(schema_table_empty)" = "t" ] && [ "$(table_exists public.tenants)" = "t" ]; then
    echo "Var olan tablolari bulunan bir veritabaninda schema_migrations bos. BOOTSTRAP_MIGRATIONS=1 scripts/run_migrations.sh komutu ile bootstrap yapin." >&2
    return 1
  fi

  local applied=0 skipped=0

  for file_path in "$MIGRATIONS_DIR"/*.up.sql; do
    [ -f "$file_path" ] || continue
    local filename
    filename=$(basename "$file_path")

    if migration_applied "$filename"; then
      echo "-> $filename (atlanildi)"
      skipped=$((skipped + 1))
      continue
    fi

    echo "-> $filename"
    apply_migration "$file_path"
    record_migration "$filename"
    applied=$((applied + 1))
  done

  echo "\033[32m[OK]\033[0m Migration islemi tamamlandi (uygulanan: $applied, atlanan: $skipped)"
}

main "$@"
