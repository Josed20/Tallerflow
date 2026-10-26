[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$verificationFailure = $null

function Assert-NativeSuccess([string]$step) {
    if ($LASTEXITCODE -ne 0) { throw "$step failed with exit code $LASTEXITCODE." }
}

# Sprint 4 verification from empty isolated volumes: the Sprint 1 and 2
# suites first, then the Sprint 4 SQL contracts, every PostgreSQL
# integration test and the browser journeys, open signup through Mailpit.
powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $root 'scripts\verify-sprint2.ps1')
Assert-NativeSuccess 'Sprint 1 and Sprint 2 suites'

$env:COMPOSE_PROJECT_NAME = 'tallerflow_sprint4_verify'
$env:COMPOSE_FILE = 'compose.yaml;compose.mail.yaml'
$env:POSTGRES_DB = 'tallerflow'
$env:POSTGRES_SUPERUSER = 'postgres'
$env:POSTGRES_SUPERUSER_PASSWORD = 'local-postgres-change-me'
$env:DB_OWNER_PASSWORD = 'local-owner-change-me'
$env:DB_MIGRATION_PASSWORD = 'local-migrator-change-me'
$env:DB_APP_PASSWORD = 'local-app-change-me'
$env:DB_BOOTSTRAP_PASSWORD = 'local-bootstrap-change-me'
$env:DB_CONSOLE_PASSWORD = 'local-console-change-me'
$env:POSTGRES_PORT = '55435'
$env:BACKEND_PORT = '18082'
$env:MAILPIT_HTTP_PORT = '18025'
$env:MAILPIT_SMTP_PORT = '11025'
$env:TF_DATABASE_URL = 'postgres://tallerflow_app:local-app-change-me@postgres:5432/tallerflow?sslmode=disable'
$env:TF_BOOTSTRAP_DATABASE_URL = 'postgres://tallerflow_bootstrap:local-bootstrap-change-me@postgres:5432/tallerflow?sslmode=disable'
$env:TF_SESSION_PEPPER = 'local-session-pepper-change-me'
$env:TF_ENVIRONMENT = 'development'
$env:TF_ALLOWED_ORIGIN = 'http://localhost:18082'
$env:TF_PUBLIC_BASE_URL = 'http://localhost:18082'
$env:TF_PASSWORD_RESET_BASE_URL = 'http://localhost:18082'
$env:TF_SIGNUP_ENABLED = 'true'
$env:TF_TRUSTED_PROXIES = '172.16.0.0/12'
$env:E2E_BASE_URL = 'http://localhost:18082'
$env:E2E_MAILPIT_URL = 'http://127.0.0.1:18025'

Push-Location $root
try {
    docker compose --profile app --profile tools down --volumes --remove-orphans
    Assert-NativeSuccess 'Initial isolated Compose cleanup'

    docker compose up -d --wait postgres
    Assert-NativeSuccess 'Database startup'
    docker compose run --rm flyway
    Assert-NativeSuccess 'Database migration'
    Get-Content database/tests/email_verification_contract.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_app -d tallerflow
    Assert-NativeSuccess 'Email verification contract tests'
    Get-Content database/tests/sprint4_michael_contract.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d tallerflow
    Assert-NativeSuccess 'Order recovery and console contract tests'

    $env:TF_TEST_DATABASE_URL = 'postgres://tallerflow_app:local-app-change-me@127.0.0.1:55435/tallerflow?sslmode=disable'
    $env:TF_TEST_BOOTSTRAP_DATABASE_URL = 'postgres://tallerflow_bootstrap:local-bootstrap-change-me@127.0.0.1:55435/tallerflow?sslmode=disable'
    $env:TF_TEST_ADMIN_DATABASE_URL = 'postgres://postgres:local-postgres-change-me@127.0.0.1:55435/tallerflow?sslmode=disable'
    $env:TF_TEST_CONSOLE_DATABASE_URL = 'postgres://tallerflow_console:local-console-change-me@127.0.0.1:55435/tallerflow?sslmode=disable'
    # Onboarding's test needs an unclaimed installation, so verify-sprint2 runs it
    # on its own database. The other packages share this one, one at a time.
    $packages = go -C backend list ./... | Where-Object { $_ -notmatch '/internal/onboarding$' }
    Assert-NativeSuccess 'Backend package listing'
    go -C backend test -count=1 -p 1 -run TestPostgresIntegration @packages
    Assert-NativeSuccess 'PostgreSQL integration tests'

    docker compose down --volumes --remove-orphans
    Assert-NativeSuccess 'Reset before browser journeys'
    docker compose --profile app up -d --build --wait
    Assert-NativeSuccess 'Application and Mailpit startup from empty volumes'
    npm --prefix frontend exec playwright install chromium
    Assert-NativeSuccess 'Playwright Chromium installation'
    # Set only now: with these, the auth integration test expects a bootstrap owner.
    $env:E2E_OWNER_EMAIL = 'owner.onboarding@tallerflow.test'
    $env:E2E_OWNER_PASSWORD = 'Permanent secure passphrase 27!'
    # Onboarding first: once signup creates a workshop the installation is claimed.
    npm --prefix frontend run test:e2e -- onboarding-flow.spec.ts
    Assert-NativeSuccess 'Playwright onboarding journey'
    npm --prefix frontend run test:e2e -- signup-flow.spec.ts
    Assert-NativeSuccess 'Playwright open signup journey'
}
catch {
    $verificationFailure = $_
}
finally {
    $ErrorActionPreference = 'Continue'
    docker compose logs --no-color --tail 200 2>&1 | Out-File -FilePath (Join-Path $root 'sprint4-compose.log') -Encoding utf8
    docker compose --profile app --profile tools down --volumes --remove-orphans
    Pop-Location
}

$ErrorActionPreference = 'Stop'
if ($null -ne $verificationFailure) { throw $verificationFailure }
