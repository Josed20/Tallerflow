#!/usr/bin/env bash
set -euo pipefail

# Emails DISK_ALERT_TO when the partition holding Docker and PostgreSQL data
# passes the threshold. It sends through the API's own SMTP settings
# (TF_SMTP_* in /opt/tallerflow/.env), so the VPS needs no other mail service.
# Pass --test to send one message now, whatever the usage.

: "${DISK_ALERT_TO:?Set the alert recipient}"
: "${TF_SMTP_HOST:?Set TF_SMTP_HOST}"
: "${TF_SMTP_USER:?Set TF_SMTP_USER}"
: "${TF_SMTP_PASSWORD:?Set TF_SMTP_PASSWORD}"
: "${TF_SMTP_FROM:?Set TF_SMTP_FROM}"

threshold=${DISK_ALERT_THRESHOLD:-80}
watched=${DISK_ALERT_PATH:-/var/lib/docker}
# While usage stays high, repeat the email at most once every 12 hours.
repeat_seconds=43200
stamp="${STATE_DIRECTORY:-/var/tmp}/disk-alert.last"
mode=${1:-check}

usage=$(df --output=pcent "$watched" | tail -1 | tr -dc '0-9')
[[ "$usage" =~ ^[0-9]+$ ]] || { echo "Could not read disk usage for $watched." >&2; exit 1; }

if [[ "$mode" != "--test" ]]; then
  if (( usage < threshold )); then
    echo "Disk $watched at ${usage}% (threshold ${threshold}%)."
    exit 0
  fi
  if [[ -f "$stamp" ]] && (( $(date +%s) - $(stat -c %Y "$stamp") < repeat_seconds )); then
    echo "Disk $watched at ${usage}%; already alerted in the last 12 hours."
    exit 0
  fi
fi

host=$(hostname)
if [[ "$mode" == "--test" ]]; then
  subject="[TallerFlow] Prueba de la alerta de disco en $host"
else
  subject="[TallerFlow] Disco al ${usage}% en $host"
fi
details=$(df -hP "$watched" | tail -1)

message=$(mktemp)
trap 'rm -f "$message"' EXIT
printf 'From: TallerFlow <%s>\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n' \
  "$TF_SMTP_FROM" "$DISK_ALERT_TO" "$subject" "$(date -R)" > "$message"
printf 'La particion de %s esta al %s%% (umbral %s%%).\r\n\r\n%s\r\n\r\nRevisa con: docker system df y df -h %s\r\n' \
  "$watched" "$usage" "$threshold" "$details" "$watched" >> "$message"

port=${TF_SMTP_PORT:-587}
scheme=smtp
[[ "$port" == 465 ]] && scheme=smtps
# Inside a quoted curl config value, backslashes and double quotes are escapes.
curl_quote() {
  local value=${1//\\/\\\\}
  printf '%s' "${value//\"/\\\"}"
}
# The credentials go through stdin so they never appear in the process list.
printf 'user = "%s:%s"\n' "$(curl_quote "$TF_SMTP_USER")" "$(curl_quote "$TF_SMTP_PASSWORD")" |
  curl --config - --silent --show-error --fail --ssl-reqd \
    --url "$scheme://$TF_SMTP_HOST:$port" \
    --mail-from "$TF_SMTP_FROM" --mail-rcpt "$DISK_ALERT_TO" \
    --upload-file "$message"

[[ "$mode" == "--test" ]] || touch "$stamp"
echo "Disk alert sent to $DISK_ALERT_TO (usage ${usage}%)."
