#!/usr/bin/env bash
set -euo pipefail

: "${BACKUP_AGE_RECIPIENT:?Set the age public recipient}"
: "${BACKUP_S3_BUCKET:?Set the external backup bucket}"
: "${AWS_ENDPOINT_URL:?Set the Cloudflare R2 S3 endpoint}"

project_dir=${1:?Pass the VPS checkout path}
cd "$project_dir"
backup_file=$(mktemp "${TMPDIR:-/tmp}/tallerflow-XXXXXXXX.dump.age")
trap 'rm -f "$backup_file"' EXIT

docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml exec -T postgres \
  sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --format=custom' \
  | age -r "$BACKUP_AGE_RECIPIENT" -o "$backup_file"

date_utc=$(date -u +%F)
aws s3 cp "$backup_file" "s3://$BACKUP_S3_BUCKET/daily/$date_utc.dump.age" --no-progress
aws s3api head-object --bucket "$BACKUP_S3_BUCKET" --key "daily/$date_utc.dump.age" >/dev/null
if [[ $(date -u +%u) == 7 ]]; then
  aws s3 cp "$backup_file" "s3://$BACKUP_S3_BUCKET/weekly/$(date -u +%G-W%V).dump.age" --no-progress
  aws s3api head-object --bucket "$BACKUP_S3_BUCKET" --key "weekly/$(date -u +%G-W%V).dump.age" >/dev/null
fi
echo "Encrypted backup uploaded for $date_utc"
