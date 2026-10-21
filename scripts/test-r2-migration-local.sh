#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
test_db="tallerflow_r2_test_$$"
cleanup() {
  docker compose exec -T postgres sh -c 'dropdb -U "$POSTGRES_USER" --if-exists "$1"' sh "$test_db" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker compose exec -T postgres sh -eu -c '
  createdb -U "$POSTGRES_USER" -O tallerflow_migrator "$1"
  psql -U "$POSTGRES_USER" -d "$1" -v ON_ERROR_STOP=1 -c "CREATE EXTENSION IF NOT EXISTS citext; CREATE EXTENSION IF NOT EXISTS pgcrypto;" >/dev/null
' sh "$test_db"

docker compose run --rm -T \
  -e "FLYWAY_URL=jdbc:postgresql://postgres:5432/$test_db" \
  flyway migrate

result=$(docker compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d "$1" -Atc "SELECT count(*) FROM stage_photos WHERE photo_data IS NULL OR object_key IS NOT NULL"' sh "$test_db")
[[ "$result" == "0" ]] || { echo "Legacy photos changed during V17 migration." >&2; exit 1; }
docker compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d "$1" -Atc "SELECT column_name FROM information_schema.columns WHERE table_name = '\''stage_photos'\'' AND column_name = '\''object_key'\''"' sh "$test_db" | grep -qx object_key
docker compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d "$1" -Atc "SELECT conname FROM pg_constraint WHERE conname = '\''stage_photos_storage_check'\''"' sh "$test_db" | grep -qx stage_photos_storage_check
echo "V17 passed on an isolated copy; legacy photos are intact."
