#!/usr/bin/env bash
set -euo pipefail

: "${BACKUP_S3_BUCKET:?Set the backup bucket}"
: "${AWS_ENDPOINT_URL:?Set the Cloudflare R2 S3 endpoint}"
: "${BACKUP_AGE_IDENTITY:?Set the path to the offline age identity}"

object_key=${1:?Pass a backup object key, for example daily/2026-10-21.dump.age}
if [[ ! "$object_key" =~ ^(daily/[0-9]{4}-[0-9]{2}-[0-9]{2}|weekly/[0-9]{4}-W[0-9]{2})\.dump\.age$ ]]; then
  echo "Invalid backup object key." >&2
  exit 1
fi

for command in aws age docker; do
  command -v "$command" >/dev/null || { echo "$command is required." >&2; exit 1; }
done
[[ -f "$BACKUP_AGE_IDENTITY" ]] || { echo "Age identity file not found." >&2; exit 1; }

archive=$(mktemp "${TMPDIR:-/tmp}/tallerflow-restore-XXXXXXXX.dump.age")
container="tallerflow-restore-$(date +%s)-$$"
cleanup() {
  docker rm -f "$container" >/dev/null 2>&1 || true
  rm -f "$archive"
}
trap cleanup EXIT

started=$(date +%s)
aws s3 cp "s3://$BACKUP_S3_BUCKET/$object_key" "$archive" --no-progress
[[ -s "$archive" ]] || { echo "Downloaded backup is empty." >&2; exit 1; }

# The temporary database has no network interface and no connection to production.
docker run -d --rm --network none --name "$container" \
  -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.6-alpine >/dev/null
for _ in {1..30}; do
  if docker exec "$container" pg_isready -U postgres -d postgres >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "$container" pg_isready -U postgres -d postgres >/dev/null
age -d -i "$BACKUP_AGE_IDENTITY" "$archive" |
  docker exec -i "$container" pg_restore -U postgres -d postgres \
    --no-owner --no-acl --exit-on-error

tables=$(docker exec "$container" psql -U postgres -d postgres -Atc \
  "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'")
[[ "$tables" =~ ^[0-9]+$ && "$tables" -gt 0 ]] || {
  echo "Restore drill failed: no public tables." >&2
  exit 1
}
for table in users workshops orders stage_photos; do
  rows=$(docker exec "$container" psql -U postgres -d postgres -Atc "SELECT count(*) FROM $table")
  echo "$table: $rows rows"
done
echo "Restore drill passed: $tables public tables from $object_key in $(($(date +%s) - started)) seconds."
