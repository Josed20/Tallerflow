# Sprint 2 José Onboarding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the web onboarding path for the first workshop OWNER, create its authenticated session, and make the empty-installation demo reproducible in CI.

**Architecture:** A new `internal/onboarding` module owns public status and HTTP transport. It reuses the existing Argon2id bootstrap and session primitives, while the API opens a second, narrowly privileged PostgreSQL connection for onboarding only. The Vue router queries onboarding availability before resolving login, and the verification script proves the flow from empty volumes.

**Tech Stack:** Go 1.27, Gin, pgx/GORM, PostgreSQL 18/Flyway, Vue 3/Pinia/Vitest, Playwright, Docker Compose, OpenAPI 3.1.

**Spec:** `docs/superpowers/specs/2026-09-27-tallerflow-sprint-2-design.md`

## Global Constraints

- PostgreSQL and Flyway remain the only schema source; never call `AutoMigrate`.
- Runtime data uses `tallerflow_app`; initial-owner writes use the separate `tallerflow_bootstrap` connection.
- Only an empty installation can create the first OWNER; concurrent requests create exactly one complete identity graph.
- The onboarding password is final (`must_change_password=false`) with `password_changed_at` populated.
- Session and CSRF tokens remain opaque, hashed at rest, same-origin and absent from browser storage and logs.
- Public POST requests validate `Origin`, enforce a bounded per-IP rate limit and return structured, non-sensitive errors.
- The CLI bootstrap keeps its temporary-password behavior unchanged.
- UI controls remain keyboard accessible, at least 44 px tall and usable at 360 px.

## Review Focus

- Two simultaneous onboarding submissions with different emails must leave exactly one user, workshop, OWNER membership, credential and audit event; the winner receives one session.
- A failure while inserting any identity row must roll back the whole onboarding transaction; a later app-role session failure must leave a valid claimed OWNER able to log in.
- Malformed, oversized and cross-origin requests must fail before hashing or database writes and never echo submitted secrets.
- A stale availability response must not bypass the database singleton; the losing POST must return the same neutral conflict response.
- Router status/network failures must not form redirect loops or hide an already authenticated session.

---

### Task 1: Freeze onboarding HTTP contract and configuration

**Files:**
- Create: `docs/contracts/openapi/sprint2/onboarding.yaml`
- Modify: `docs/contracts/openapi.yaml`
- Modify: `backend/platform/config/config.go`
- Test: `backend/platform/config/config_test.go`

**Interfaces:**
- Produces: `config.Config.BootstrapDatabaseURL string`; `GET /api/v1/onboarding/status`; `POST /api/v1/onboarding/workshop` request and response schemas.

- [ ] Write failing configuration tests requiring `TF_BOOTSTRAP_DATABASE_URL` without leaking its value and covering a valid separate URL.
- [ ] Run `go test ./platform/config -run Bootstrap -v`; expect failure because the field is absent.
- [ ] Add `BootstrapDatabaseURL` loading and validation, then run the focused and full config tests green.
- [ ] Add the standalone OpenAPI 3.1 onboarding fragment and merge the same paths/schemas into the root contract.
- [ ] Validate both contracts with `npx --yes @redocly/cli@2.11.1 lint` and commit.

### Task 2: Make initial-owner persistence support a permanent credential and browser session

**Files:**
- Modify: `backend/internal/auth/bootstrap.go`
- Modify: `backend/internal/auth/bootstrap_postgres.go`
- Modify: `backend/internal/auth/bootstrap_test.go`
- Modify: `backend/internal/auth/session.go`
- Modify: `backend/internal/auth/session_test.go`

**Interfaces:**
- Produces: `BootstrapService.CreateWebOwner(ctx, BootstrapInput, SessionMetadata, *SessionService) (WebBootstrapResult, error)`; CLI `CreateOwner` behavior remains unchanged.
- Produces: `SessionService.Prepare(userID, metadata) (PreparedSession, error)` where `PreparedSession` contains raw token, raw CSRF token and the hashed `NewSession` row.

- [ ] Add failing tests for permanent credentials, `password_changed_at`, session persistence, identity rollback and unchanged CLI semantics.
- [ ] Run focused auth tests and confirm each fails for missing web-owner/session behavior.
- [ ] Extend the bootstrap transaction port and pgx adapter to insert the credential timestamp, then create the browser session through the existing app-role session repository after the identity transaction commits.
- [ ] Refactor `SessionService` around `Prepare` while preserving existing `Create`, then make all auth tests green.
- [ ] Run `go test ./internal/auth/...` and commit.

### Task 3: Implement onboarding status, service, handler and security limits

**Files:**
- Create: `backend/internal/onboarding/service.go`
- Create: `backend/internal/onboarding/service_test.go`
- Create: `backend/internal/onboarding/postgres.go`
- Create: `backend/internal/onboarding/handler.go`
- Create: `backend/internal/onboarding/handler_test.go`
- Create: `backend/internal/onboarding/routes.go`
- Create: `backend/platform/security/ratelimit_generic.go`
- Create: `backend/platform/security/ratelimit_generic_test.go`

**Interfaces:**
- Produces: `Service.Status(context.Context) (Status, error)` and `Service.Create(context.Context, Input, auth.SessionMetadata) (Result, error)`.
- Produces: `RegisterRoutes(gin.IRouter, *Handler)` for both public endpoints.

- [ ] Add failing service tests for available/claimed status, normalized input, weak input, concurrent winner, atomic rollback and safe claimed errors.
- [ ] Add failing handler tests for status shape, 201 cookie/session response, origin rejection, 409, 422 field details, 429 and secret-free errors.
- [ ] Add failing generic limiter tests for per-IP boundaries, expiry and bounded capacity.
- [ ] Implement the minimal service, pgx status reader, limiter and handler; run focused tests after each RED→GREEN cycle.
- [ ] Run `go test ./internal/onboarding ./platform/security` and commit.

### Task 4: Compose the restricted onboarding module into the API

**Files:**
- Modify: `backend/cmd/api/app.go`
- Modify: `backend/cmd/api/app_test.go`
- Modify: `compose.yaml`
- Modify: `.env.example`

**Interfaces:**
- Consumes: `config.Config.BootstrapDatabaseURL`, `auth.NewPostgresBootstrapStore`, `onboarding.NewPostgresStatusStore`, `onboarding.RegisterRoutes`.
- Produces: API startup with independently closed app and bootstrap pools.

- [ ] Add failing app tests proving onboarding routes are registered, both database URLs use separate openers and both resources close once.
- [ ] Run `go test ./cmd/api -v`; expect failures for absent composition.
- [ ] Add the bootstrap pool dependency and onboarding route registration without granting bootstrap rights to the normal GORM connection.
- [ ] Pass `TF_BOOTSTRAP_DATABASE_URL` to the backend Compose service and make app tests/full backend tests green.
- [ ] Commit.

### Task 5: Build the accessible onboarding frontend and session handoff

**Files:**
- Create: `frontend/src/modules/onboarding/api.ts`
- Create: `frontend/src/modules/onboarding/onboarding.store.ts`
- Create: `frontend/src/modules/onboarding/OnboardingView.vue`
- Create: `frontend/src/modules/onboarding/OnboardingView.test.ts`
- Create: `frontend/src/modules/onboarding/onboarding.store.test.ts`
- Modify: `frontend/src/modules/session/session.store.ts`
- Modify: `frontend/src/modules/session/session.store.test.ts`
- Modify: `frontend/src/styles/main.css`

**Interfaces:**
- Produces: `useOnboardingStore().loadStatus(force?)` and `.create(input)`; successful creation hands the returned session to `useSessionStore().acceptSession`.

- [ ] Add failing store tests for cached availability, forced refresh, session handoff and API errors.
- [ ] Add failing component tests for required fields, email, 12-character password, confirmation, first-error focus, busy double-submit guard and server errors.
- [ ] Implement API/store/session handoff and the form with accessible status/errors and 360 px styling.
- [ ] Run focused tests and the complete frontend unit suite green.
- [ ] Commit.

### Task 6: Route empty and claimed installations correctly

**Files:**
- Modify: `frontend/src/app/router.ts`
- Create: `frontend/src/app/router.test.ts`

**Interfaces:**
- Consumes: onboarding status and existing session restoration.
- Produces: empty `/login` → `/onboarding`; claimed `/onboarding` → `/login`; authenticated users → `/app`; status failures do not loop.

- [ ] Add failing memory-router tests for empty, claimed, authenticated and status-error cases.
- [ ] Extract a testable router factory and implement the guards without changing password-rotation behavior.
- [ ] Run router tests and the complete frontend suite green.
- [ ] Commit.

### Task 7: Prove PostgreSQL concurrency and empty-volume browser journey

**Files:**
- Create: `backend/internal/onboarding/postgres_integration_test.go`
- Create: `frontend/e2e/onboarding-flow.spec.ts`
- Create: `scripts/verify-sprint2.ps1`
- Create: `scripts/verify-sprint2.sh`
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`

**Interfaces:**
- Consumes: all prior onboarding endpoints and UI.
- Produces: repeatable Sprint 2 José verification from empty volumes and CI evidence.

- [ ] Add a live PostgreSQL test that races two web-owner transactions and asserts exactly one complete graph and one session.
- [ ] Add Playwright coverage for empty redirect, validation, successful creation, authenticated `/app`, claimed redirect, browser-storage absence and 360 px overflow.
- [ ] Create PowerShell and POSIX verification scripts that run race/unit/build/OpenAPI/SQL/PostgreSQL/E2E checks with isolated Compose names and volumes.
- [ ] Update CI to run `verify-sprint2.sh`, upload logs/evidence on failure, and document the native commands.
- [ ] Run `scripts/verify-sprint2.ps1` from empty volumes and commit.

### Task 8: Final integration readiness and branch verification

**Files:**
- Modify only files required by findings from the whole-branch review.

**Interfaces:**
- Consumes: Tasks 1–7.
- Produces: a green José branch ready for PR and later consolidation with recovery, orders and team fragments.

- [ ] Run `go test -race ./...`, frontend unit/build, OpenAPI lint and the empty-volume verification script.
- [ ] Compare J-01–J-14 against the spec; record that four-module consolidation and the final cross-team demo remain integration-time gates until the other three branches exist.
- [ ] Generate the review package, perform one fresh whole-branch review and fix every Critical/Important finding with RED→GREEN tests.
- [ ] Run the full verification again and commit any review fixes.
