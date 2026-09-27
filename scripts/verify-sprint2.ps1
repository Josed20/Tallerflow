[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$backendPath = Join-Path $root 'backend'
$verificationFailure = $null

function Assert-NativeSuccess([string]$step) {
    if ($LASTEXITCODE -ne 0) { throw "$step failed with exit code $LASTEXITCODE." }
}

$env:COMPOSE_PROJECT_NAME = 'tallerflow_sprint2_verify'
$env:POSTGRES_DB = 'tallerflow'
$env:POSTGRES_SUPERUSER = 'postgres'
$env:POSTGRES_SUPERUSER_PASSWORD = 'local-postgres-change-me'
$env:DB_OWNER_PASSWORD = 'local-owner-change-me'
$env:DB_MIGRATION_PASSWORD = 'local-migrator-change-me'
$env:DB_APP_PASSWORD = 'local-app-change-me'
$env:DB_BOOTSTRAP_PASSWORD = 'local-bootstrap-change-me'
$env:POSTGRES_PORT = '55434'
$env:BACKEND_PORT = '18081'
$env:TF_DATABASE_URL = 'postgres://tallerflow_app:local-app-change-me@postgres:5432/tallerflow?sslmode=disable'
$env:TF_BOOTSTRAP_DATABASE_URL = 'postgres://tallerflow_bootstrap:local-bootstrap-change-me@postgres:5432/tallerflow?sslmode=disable'
$env:TF_SESSION_PEPPER = 'local-session-pepper-change-me'
$env:TF_ENVIRONMENT = 'development'
$env:TF_ALLOWED_ORIGIN = 'http://localhost:18081'
$env:TF_TRUSTED_PROXIES = '172.16.0.0/12'
$env:E2E_BASE_URL = 'http://localhost:18081'
$env:E2E_OWNER_EMAIL = 'owner.onboarding@tallerflow.test'
$env:E2E_OWNER_PASSWORD = 'Permanent secure passphrase 27!'

Push-Location $root
try {
    powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $root 'scripts\verify-sprint1.ps1')
    Assert-NativeSuccess 'Sprint 1 authentication regression suite'

    docker compose --profile app --profile tools down --volumes --remove-orphans
    Assert-NativeSuccess 'Initial isolated Compose cleanup'

    docker run --rm --volume "${backendPath}:/src" --workdir /src golang:1.27-bookworm go test -race ./...
    Assert-NativeSuccess 'Backend race tests'
    docker run --rm --volume "${backendPath}:/src" --workdir /src golang:1.27-bookworm go vet ./...
    Assert-NativeSuccess 'Backend static analysis'
    npm --prefix frontend ci --no-audit
    Assert-NativeSuccess 'Frontend dependency installation'
    npm --prefix frontend test
    Assert-NativeSuccess 'Frontend unit tests'
    npm --prefix frontend run build
    Assert-NativeSuccess 'Frontend production build'
    npx --yes '@redocly/cli@2.11.1' lint docs/contracts/openapi.yaml docs/contracts/openapi/sprint2/onboarding.yaml
    Assert-NativeSuccess 'OpenAPI lint'

    docker compose up -d --wait postgres flyway
    Assert-NativeSuccess 'Database migration startup'
    docker compose run --rm flyway validate
    Assert-NativeSuccess 'Flyway validation'
    Get-Content database/tests/constraints.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow
    Assert-NativeSuccess 'Database constraint tests'
    Get-Content database/tests/identity_contract.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow
    Assert-NativeSuccess 'Identity contract tests'
    Get-Content database/tests/rls_setup.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow
    Assert-NativeSuccess 'RLS fixture setup'
    Get-Content database/tests/rls.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_app -d tallerflow
    Assert-NativeSuccess 'Tenant isolation SQL tests'

    docker compose down --volumes --remove-orphans
    Assert-NativeSuccess 'Reset before concurrent onboarding test'
    docker compose up -d --wait postgres flyway
    Assert-NativeSuccess 'Fresh integration database startup'
    $env:TF_TEST_DATABASE_URL = 'postgres://tallerflow_app:local-app-change-me@127.0.0.1:55434/tallerflow?sslmode=disable'
    $env:TF_TEST_BOOTSTRAP_DATABASE_URL = 'postgres://tallerflow_bootstrap:local-bootstrap-change-me@127.0.0.1:55434/tallerflow?sslmode=disable'
    $env:TF_TEST_ADMIN_DATABASE_URL = 'postgres://postgres:local-postgres-change-me@127.0.0.1:55434/tallerflow?sslmode=disable'
    go -C backend test ./internal/onboarding -run TestPostgresIntegration -v
    Assert-NativeSuccess 'Concurrent onboarding PostgreSQL integration test'

    docker compose down --volumes --remove-orphans
    Assert-NativeSuccess 'Reset before browser journey'
    docker compose --profile app up -d --build --wait
    Assert-NativeSuccess 'Application startup from empty volumes'
    npm --prefix frontend exec playwright install chromium
    Assert-NativeSuccess 'Playwright Chromium installation'
    npm --prefix frontend run test:e2e -- onboarding-flow.spec.ts
    Assert-NativeSuccess 'Playwright onboarding journey'
}
catch {
    $verificationFailure = $_
}
finally {
    $ErrorActionPreference = 'Continue'
    docker compose logs --no-color --tail 200 2>&1 | Out-File -FilePath (Join-Path $root 'sprint2-compose.log') -Encoding utf8
    docker compose --profile app --profile tools down --volumes --remove-orphans
    Pop-Location
}

$ErrorActionPreference = 'Stop'
if ($null -ne $verificationFailure) { throw $verificationFailure }
