#!/usr/bin/env bash
set -euo pipefail

project_dir=${1:?Pass the VPS checkout path}
health_url=${2:?Pass the public HTTPS readiness URL}
cd "$project_dir"

if [[ -n $(git status --porcelain) ]]; then
  echo 'The VPS checkout has local changes; deployment stopped.' >&2
  exit 1
fi

previous=$(git rev-parse HEAD)
git switch main
git pull --ff-only origin main

# compose.tunnel.yaml keeps Caddy unpublished: Docker bypasses ufw for
# published ports, so without it port 8080 would be open in plain HTTP.
deploy() {
  docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml --profile app up -d --build --wait
  curl --fail --silent --show-error --retry 8 --retry-delay 3 "$health_url" >/dev/null
}

if ! deploy; then
  echo 'Readiness failed; rebuilding the previous commit.' >&2
  git switch --detach "$previous"
  deploy
  exit 1
fi

echo "Deployed $(git rev-parse --short HEAD)"
