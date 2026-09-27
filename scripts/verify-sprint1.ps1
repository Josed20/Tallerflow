[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$backendPath = Join-Path $root 'backend'
$ownerEmail = 'owner.e2e@tallerflow.test'
$ownerPassword = 'Temporary secure passphrase 9!'
$replacementPassword = 'Replacement secure passphrase 10!'
$verificationFailure = $null

function Assert-NativeSuccess([string]$step) {
    if ($LASTEXITCODE -ne 0) {
        throw "$step failed with exit code $LASTEXITCODE."
    }
}

$env:COMPOSE_PROJECT_NAME = 'tallerflow_sprint1_verify'
$env:POSTGRES_DB = 'tallerflow'
$env:POSTGRES_PORT = '55432'
$env:BACKEND_PORT = '18080'
$env:TF_ALLOWED_ORIGIN = 'http://localhost:18080'
$env:TF_TRUSTED_PROXIES = '172.16.0.0/12'
$env:E2E_BASE_URL = 'http://localhost:18080'
$env:E2E_OWNER_EMAIL = $ownerEmail
$env:E2E_OWNER_PASSWORD = $ownerPassword
$env:E2E_NEW_PASSWORD = $replacementPassword

Push-Location $root
try {
    docker compose --profile app --profile tools down --volumes --remove-orphans
    Assert-NativeSuccess 'Initial isolated Compose cleanup'

    docker run --rm --volume "${backendPath}:/src" --workdir /src golang:1.27-bookworm go test -race ./...
    Assert-NativeSuccess 'Backend race tests'
    npm --prefix frontend ci --no-audit
    Assert-NativeSuccess 'Frontend dependency installation'
    npm --prefix frontend test
    Assert-NativeSuccess 'Frontend unit tests'
    npm --prefix frontend run build
    Assert-NativeSuccess 'Frontend production build'

    docker compose --profile tools build bootstrap
    Assert-NativeSuccess 'Bootstrap image build'

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

    $ownerPassword | docker compose --profile tools run --rm -T bootstrap --password-stdin --email $ownerEmail --name 'Owner E2E' --workshop 'Taller E2E'
    Assert-NativeSuccess 'Initial owner bootstrap'
    $previousErrorAction = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $secondBootstrap = $ownerPassword | docker compose --profile tools run --rm -T bootstrap --password-stdin --email $ownerEmail --name 'Owner E2E' --workshop 'Taller E2E' 2>&1 | ForEach-Object { "$_" }
    $secondExit = $LASTEXITCODE
    $ErrorActionPreference = $previousErrorAction
    if ($secondExit -eq 0 -or ($secondBootstrap -join "`n") -notmatch 'BOOTSTRAP_ALREADY_EXISTS') {
        throw 'Second bootstrap did not fail with BOOTSTRAP_ALREADY_EXISTS.'
    }

    $env:TF_TEST_DATABASE_URL = 'postgres://tallerflow_app:local-app-change-me@127.0.0.1:55432/tallerflow?sslmode=disable'
    go -C backend test ./internal/auth -run TestPostgresIntegration -v
    Assert-NativeSuccess 'Live PostgreSQL integration tests'

    docker compose --profile app up -d --build --wait
    Assert-NativeSuccess 'Application Compose startup'
    $ErrorActionPreference = 'Continue'
    $bootstrapHelp = docker compose --profile tools run --rm -T bootstrap -h 2>&1 | ForEach-Object { "$_" }
    $bootstrapHelpExit = $LASTEXITCODE
    $ErrorActionPreference = $previousErrorAction
    if ($bootstrapHelpExit -eq 0 -or ($bootstrapHelp -join "`n") -notmatch 'password-stdin') {
        throw 'The backend image did not expose the bootstrap command.'
    }
    npm --prefix frontend exec playwright install chromium
    Assert-NativeSuccess 'Playwright Chromium installation'
    npm --prefix frontend run test:e2e
    Assert-NativeSuccess 'Playwright authentication journey'
}
catch {
    $verificationFailure = $_
}
finally {
    $ErrorActionPreference = 'Continue'
    docker compose logs --no-color --tail 200 2>&1 | Out-File -FilePath (Join-Path $root 'sprint1-compose.log') -Encoding utf8
    docker compose --profile app --profile tools down --volumes --remove-orphans
    Pop-Location
}

$ErrorActionPreference = 'Stop'
if ($null -ne $verificationFailure) {
    throw $verificationFailure
}
