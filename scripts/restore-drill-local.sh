#!/usr/bin/env bash
set -euo pipefail

project_dir=${1:-.}
cd "$project_dir"

docker compose --profile app exec -T postgres sh -eu -c '
  restore_db="tallerflow_restore_drill_$(date +%s)_$$"
  archive="/tmp/${restore_db}.dump"
  cleanup() {
    dropdb -U "$POSTGRES_USER" --if-exists "$restore_db" >/dev/null 2>&1 || true
    rm -f "$archive"
  }
  trap cleanup EXIT

  pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --format=custom -f "$archive"
  createdb -U "$POSTGRES_USER" "$restore_db"
  pg_restore -U "$POSTGRES_USER" -d "$restore_db" --no-owner --no-acl --exit-on-error "$archive"
  source_tables=$(psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "SELECT count(*) FROM information_schema.tables WHERE table_schema = '\''public'\'' AND table_type = '\''BASE TABLE'\''")
  restored_tables=$(psql -U "$POSTGRES_USER" -d "$restore_db" -Atc "SELECT count(*) FROM information_schema.tables WHERE table_schema = '\''public'\'' AND table_type = '\''BASE TABLE'\''")
  if [ "$source_tables" -eq 0 ] || [ "$source_tables" -ne "$restored_tables" ]; then
    echo "Restore drill failed: source has $source_tables tables, restored database has $restored_tables." >&2
    exit 1
  fi
  for table in users workshops orders stage_photos; do
    source_rows=$(psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "SELECT count(*) FROM $table")
    restored_rows=$(psql -U "$POSTGRES_USER" -d "$restore_db" -Atc "SELECT count(*) FROM $table")
    if [ "$source_rows" -ne "$restored_rows" ]; then
      echo "Restore drill failed: $table has $source_rows source rows and $restored_rows restored rows." >&2
      exit 1
    fi
  done
  echo "Restore drill passed: $restored_tables public tables and key data counts match in an isolated temporary database."
'
