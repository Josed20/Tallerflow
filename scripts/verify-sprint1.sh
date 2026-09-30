#!/usr/bin/env sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
owner_email=owner.e2e@tallerflow.test
owner_password='Temporary secure passphrase 9!'
replacement_password='Replacement secure passphrase 10!'
log_file="$root/sprint1-compose.log"

export POSTGRES_DB=tallerflow
export COMPOSE_PROJECT_NAME=tallerflow_sprint1_verify
export POSTGRES_SUPERUSER=postgres
export POSTGRES_SUPERUSER_PASSWORD=local-postgres-change-me
export DB_OWNER_PASSWORD=local-owner-change-me
export DB_MIGRATION_PASSWORD=local-migrator-change-me
export DB_APP_PASSWORD=local-app-change-me
export DB_BOOTSTRAP_PASSWORD=local-bootstrap-change-me
export POSTGRES_PORT=55432
export BACKEND_PORT=18080
export TF_DATABASE_URL='postgres://tallerflow_app:local-app-change-me@postgres:5432/tallerflow?sslmode=disable'
export TF_BOOTSTRAP_DATABASE_URL='postgres://tallerflow_bootstrap:local-bootstrap-change-me@postgres:5432/tallerflow?sslmode=disable'
export TF_SESSION_PEPPER=local-session-pepper-change-me
export TF_ENVIRONMENT=development
export TF_ALLOWED_ORIGIN=http://localhost:18080
export TF_TRUSTED_PROXIES=172.16.0.0/12
export E2E_BASE_URL=http://localhost:18080
export E2E_OWNER_EMAIL=$owner_email
export E2E_OWNER_PASSWORD=$owner_password
export E2E_NEW_PASSWORD=$replacement_password

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
docker compose --profile tools build bootstrap

docker compose up -d --wait postgres flyway
docker compose run --rm flyway validate
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow < database/tests/constraints.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow < database/tests/identity_contract.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow < database/tests/rls_setup.sql
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_app -d tallerflow < database/tests/rls.sql

printf '%s\n' "$owner_password" | docker compose --profile tools run --rm -T bootstrap --password-stdin --email "$owner_email" --name 'Owner E2E' --workshop 'Taller E2E'
set +e
second_bootstrap=$(printf '%s\n' "$owner_password" | docker compose --profile tools run --rm -T bootstrap --password-stdin --email "$owner_email" --name 'Owner E2E' --workshop 'Taller E2E' 2>&1)
second_exit=$?
set -e
if [ "$second_exit" -eq 0 ] || ! printf '%s' "$second_bootstrap" | grep -q 'BOOTSTRAP_ALREADY_EXISTS'; then
  echo 'Second bootstrap did not fail with BOOTSTRAP_ALREADY_EXISTS.' >&2
  exit 1
fi

export TF_TEST_DATABASE_URL='postgres://tallerflow_app:local-app-change-me@127.0.0.1:55432/tallerflow?sslmode=disable'
go -C backend test ./internal/auth -run TestPostgresIntegration -v

docker compose --profile app up -d --build --wait
set +e
bootstrap_help=$(docker compose --profile tools run --rm -T bootstrap -h 2>&1)
bootstrap_help_exit=$?
set -e
if [ "$bootstrap_help_exit" -eq 0 ] || ! printf '%s' "$bootstrap_help" | grep -q 'password-stdin'; then
  echo 'The backend image did not expose the bootstrap command.' >&2
  exit 1
fi
npm --prefix frontend exec playwright install chromium
npm --prefix frontend run test:e2e -- auth-flow.spec.ts
