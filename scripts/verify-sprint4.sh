#!/usr/bin/env sh
set -eu

# Sprint 4 verification from empty isolated volumes: the Sprint 1 and 2
# suites first, then the Sprint 4 SQL contracts, every PostgreSQL
# integration test and the browser journeys, open signup through Mailpit.
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
log_file="$root/sprint4-compose.log"

"$root/scripts/verify-sprint2.sh"

export COMPOSE_PROJECT_NAME=tallerflow_sprint4_verify
export COMPOSE_FILE=compose.yaml:compose.mail.yaml
export POSTGRES_DB=tallerflow
export POSTGRES_SUPERUSER=postgres
export POSTGRES_SUPERUSER_PASSWORD=local-postgres-change-me
export DB_OWNER_PASSWORD=local-owner-change-me
export DB_MIGRATION_PASSWORD=local-migrator-change-me
export DB_APP_PASSWORD=local-app-change-me
export DB_BOOTSTRAP_PASSWORD=local-bootstrap-change-me
export DB_CONSOLE_PASSWORD=local-console-change-me
export POSTGRES_PORT=55435
export BACKEND_PORT=18082
export MAILPIT_HTTP_PORT=18025
export MAILPIT_SMTP_PORT=11025
export TF_DATABASE_URL='postgres://tallerflow_app:local-app-change-me@postgres:5432/tallerflow?sslmode=disable'
export TF_BOOTSTRAP_DATABASE_URL='postgres://tallerflow_bootstrap:local-bootstrap-change-me@postgres:5432/tallerflow?sslmode=disable'
export TF_SESSION_PEPPER=local-session-pepper-change-me
export TF_ENVIRONMENT=development
export TF_ALLOWED_ORIGIN=http://localhost:18082
export TF_PUBLIC_BASE_URL=http://localhost:18082
export TF_PASSWORD_RESET_BASE_URL=http://localhost:18082
export TF_SIGNUP_ENABLED=true
export TF_TRUSTED_PROXIES=172.16.0.0/12
export E2E_BASE_URL=http://localhost:18082
export E2E_MAILPIT_URL=http://127.0.0.1:18025

cleanup() {
  docker compose logs --no-color --tail 200 >"$log_file" 2>&1 || true
  docker compose --profile app --profile tools down --volumes --remove-orphans || true
}
trap cleanup EXIT INT TERM

cd "$root"
docker compose --profile app --profile tools down --volumes --remove-orphans

docker compose up -d --wait postgres
docker compose run --rm flyway
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_app -d tallerflow < database/tests/email_verification_contract.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d tallerflow < database/tests/sprint4_michael_contract.sql

export TF_TEST_DATABASE_URL='postgres://tallerflow_app:local-app-change-me@127.0.0.1:55435/tallerflow?sslmode=disable'
export TF_TEST_BOOTSTRAP_DATABASE_URL='postgres://tallerflow_bootstrap:local-bootstrap-change-me@127.0.0.1:55435/tallerflow?sslmode=disable'
export TF_TEST_ADMIN_DATABASE_URL='postgres://postgres:local-postgres-change-me@127.0.0.1:55435/tallerflow?sslmode=disable'
export TF_TEST_CONSOLE_DATABASE_URL='postgres://tallerflow_console:local-console-change-me@127.0.0.1:55435/tallerflow?sslmode=disable'
# Onboarding's test needs an unclaimed installation, so verify-sprint2 runs it
# on its own database. The other packages share this one, one at a time.
packages=$(go -C backend list ./... | grep -v '/internal/onboarding$')
# shellcheck disable=SC2086
go -C backend test -count=1 -p 1 -run TestPostgresIntegration $packages

docker compose down --volumes --remove-orphans
docker compose --profile app up -d --build --wait
npm --prefix frontend exec playwright install chromium
# Set only now: with these, the auth integration test expects a bootstrap owner.
export E2E_OWNER_EMAIL=owner.onboarding@tallerflow.test
export E2E_OWNER_PASSWORD='Permanent secure passphrase 27!'
# Onboarding first: once signup creates a workshop the installation is claimed.
npm --prefix frontend run test:e2e -- onboarding-flow.spec.ts
npm --prefix frontend run test:e2e -- signup-flow.spec.ts
