#!/usr/bin/env sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
log_file="$root/sprint2-compose.log"

export COMPOSE_PROJECT_NAME=tallerflow_sprint2_verify
export POSTGRES_DB=tallerflow
export POSTGRES_SUPERUSER=postgres
export POSTGRES_SUPERUSER_PASSWORD=local-postgres-change-me
export DB_OWNER_PASSWORD=local-owner-change-me
export DB_MIGRATION_PASSWORD=local-migrator-change-me
export DB_APP_PASSWORD=local-app-change-me
export DB_BOOTSTRAP_PASSWORD=local-bootstrap-change-me
export POSTGRES_PORT=55434
export BACKEND_PORT=18081
export TF_DATABASE_URL='postgres://tallerflow_app:local-app-change-me@postgres:5432/tallerflow?sslmode=disable'
export TF_BOOTSTRAP_DATABASE_URL='postgres://tallerflow_bootstrap:local-bootstrap-change-me@postgres:5432/tallerflow?sslmode=disable'
export TF_SESSION_PEPPER=local-session-pepper-change-me
export TF_ENVIRONMENT=development
export TF_ALLOWED_ORIGIN=http://localhost:18081
export TF_TRUSTED_PROXIES=172.16.0.0/12
export E2E_BASE_URL=http://localhost:18081
export E2E_OWNER_EMAIL=owner.onboarding@tallerflow.test
export E2E_OWNER_PASSWORD='Permanent secure passphrase 27!'

cleanup() {
  docker compose logs --no-color --tail 200 >"$log_file" 2>&1 || true
  docker compose --profile app --profile tools down --volumes --remove-orphans || true
}
trap cleanup EXIT INT TERM

cd "$root"
docker compose --profile app --profile tools down --volumes --remove-orphans
go -C backend test -race ./...
npm --prefix frontend ci --no-audit
npm --prefix frontend test
npm --prefix frontend run build
npx --yes '@redocly/cli@2.11.1' lint docs/contracts/openapi.yaml docs/contracts/openapi/sprint2/onboarding.yaml

docker compose up -d --wait postgres flyway
docker compose run --rm flyway validate
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow < database/tests/constraints.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow < database/tests/identity_contract.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow < database/tests/rls_setup.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_app -d tallerflow < database/tests/rls.sql

docker compose down --volumes --remove-orphans
docker compose up -d --wait postgres flyway
export TF_TEST_DATABASE_URL='postgres://tallerflow_app:local-app-change-me@127.0.0.1:55434/tallerflow?sslmode=disable'
export TF_TEST_BOOTSTRAP_DATABASE_URL='postgres://tallerflow_bootstrap:local-bootstrap-change-me@127.0.0.1:55434/tallerflow?sslmode=disable'
export TF_TEST_ADMIN_DATABASE_URL='postgres://postgres:local-postgres-change-me@127.0.0.1:55434/tallerflow?sslmode=disable'
go -C backend test ./internal/onboarding -run TestPostgresIntegration -v

docker compose down --volumes --remove-orphans
docker compose --profile app up -d --build --wait
npm --prefix frontend exec playwright install chromium
npm --prefix frontend run test:e2e -- onboarding-flow.spec.ts
