#!/usr/bin/env bash
set -euo pipefail

: "${BACKUP_AGE_RECIPIENT:?Set the age public recipient}"
: "${BACKUP_S3_BUCKET:?Set the external backup bucket}"

project_dir=${1:?Pass the VPS checkout path}
cd "$project_dir"
backup_file=$(mktemp "${TMPDIR:-/tmp}/tallerflow-XXXXXXXX.dump.age")
trap 'rm -f "$backup_file"' EXIT

docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml exec -T postgres \
  sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --format=custom' \
  | age -r "$BACKUP_AGE_RECIPIENT" -o "$backup_file"

date_utc=$(date -u +%F)
aws s3 cp "$backup_file" "s3://$BACKUP_S3_BUCKET/daily/$date_utc.dump.age" --no-progress
if [[ $(date -u +%u) == 7 ]]; then
  aws s3 cp "$backup_file" "s3://$BACKUP_S3_BUCKET/weekly/$(date -u +%G-W%V).dump.age" --no-progress
fi
echo "Encrypted backup uploaded for $date_utc"
