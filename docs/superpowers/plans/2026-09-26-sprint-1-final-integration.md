# TallerFlow Sprint 1 Final Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate the four Sprint 1 branches into a secure, reproducible Vue-to-Go-to-PostgreSQL flow and land the verified result on `main`.

**Architecture:** Continue from `codex/s1-sprint1-final`, merge Lucero's frontend, then complete one Gin composition root backed by PostgreSQL/GORM, transaction-safe authentication and Michael's tenant runner. Package API and bootstrap in one backend image, verify the user journey through Playwright and Compose, and fast-forward `main` only after every gate passes.

**Tech Stack:** Go 1.27.1, Gin 1.12, GORM 1.31, pgx 5.11, PostgreSQL 18.6, Flyway 13.7, Vue 3.5, Pinia 3, Vite 6, Vitest 3, Playwright, Docker Compose and GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-09-26-sprint-1-final-integration-design.md`

## Global Constraints

- Preserve all contributor branches and history; no force push or history rewrite.
- Do not touch the dirty Jose worktree or untracked files in the root checkout.
- `tallerflow_migrator`, `tallerflow_app` and `tallerflow_bootstrap` are fixed role names; only credentials and URLs are configurable.
- Flyway is the only schema manager; never call GORM `AutoMigrate`.
- The runtime API uses the application role; bootstrap uses its separate restricted role.
- Never log or persist raw passwords, session tokens, CSRF tokens, password hashes or database URLs.
- Sprint 1 supports exactly one active workshop membership per authenticated user.
- Tenant identity comes from restored session state, never request headers or route parameters.
- Domain, VPS, external backups, customers, orders and production tracking remain out of scope.

## Review Focus

- A login verified with an old password must not create a session after a concurrent password change; Task 4 adds the transaction/race tests.
- Rotating email addresses from one client IP must still reach `429` before additional Argon2 work; Task 5 adds the IP-bucket tests.
- Missing, malformed or cross-tenant session state must never expose a workshop; Task 6 adds middleware and tenant-route tests.
- Bootstrap must not leak its password through arguments, environment, Compose output or logs; Task 7 adds container and command checks.
- Frontend restoration must distinguish anonymous, forced-password-change and fully authenticated states without browser storage; Task 8 adds store and Playwright coverage.

---

### Task 1: Merge Lucero and establish a green frontend baseline

**Files:**
- Merge: `origin/feat/s1-lucero-frontend-shell`
- Verify: `frontend/package.json`
- Verify: `frontend/src/modules/session/session.store.ts`
- Verify: `frontend/src/modules/session/session.store.test.ts`

**Interfaces:**
- Consumes: backend contract in `docs/contracts/openapi.yaml`.
- Produces: the Vue shell and its existing `/api/v1/auth/*`, `/api/v1/me` and `/api/v1/workshops/current` client calls for later tasks.

- [ ] **Step 1: Merge Lucero's reviewed remote head**

Run: `git merge --no-ff origin/feat/s1-lucero-frontend-shell -m "merge: integrate Sprint 1 frontend shell"`
Expected: clean merge; no files from the root checkout or Jose worktree appear.

- [ ] **Step 2: Install the locked frontend dependencies**

Run: `cd frontend; npm ci`
Expected: exit 0 and no lockfile changes.

- [ ] **Step 3: Run Lucero's tests and production build**

Run: `cd frontend; npm test; npm run build`
Expected: all Vitest tests pass and Vite produces `dist/`.

- [ ] **Step 4: Confirm the branch is clean before integration edits**

Run: `git status --short`
Expected: no tracked modifications; ignored `node_modules/` and `dist/` do not appear.

### Task 2: Validate runtime configuration and database lifecycle

**Files:**
- Modify: `backend/platform/config/config.go`
- Modify: `backend/platform/config/config_test.go`
- Modify: `backend/platform/database/database.go`
- Create: `backend/platform/database/database_test.go`

**Interfaces:**
- Consumes: `TF_DATABASE_URL`, `TF_SESSION_PEPPER`, `TF_ALLOWED_ORIGIN`, `TF_ENVIRONMENT`, `TF_HTTP_ADDRESS` and `TF_TRUSTED_PROXIES`.
- Produces: `config.Config.TrustedProxies []string`, `database.Open(string) (*gorm.DB, error)`, `database.Ping(context.Context, *gorm.DB) error` and `database.Close(*gorm.DB) error`.

- [ ] **Step 1: Write failing config and lifecycle tests**

Add `TestLoadRequiresAllowedOrigin`, `TestLoadRejectsInvalidEnvironment`, `TestLoadParsesTrustedProxies`, `TestPingUsesUnderlyingSQLConnection` and `TestCloseUsesUnderlyingSQLConnection`. Assert production rejects a non-HTTPS origin and defaults trusted proxies to `172.16.0.0/12` only in container-oriented configuration.

- [ ] **Step 2: Run the focused tests and observe failure**

Run: `cd backend; go test ./platform/config ./platform/database -run 'TestLoad|TestPing|TestClose' -v`
Expected: FAIL because validation and lifecycle functions are missing.

- [ ] **Step 3: Implement validated configuration and DB lifecycle**

Keep errors limited to variable names. `Ping` and `Close` must call `db.DB()` and wrap failures without including the URL.

- [ ] **Step 4: Run focused and package tests**

Run: `cd backend; go test ./platform/config ./platform/database`
Expected: PASS.

- [ ] **Step 5: Commit the runtime foundation**

Run: `git add backend/platform/config backend/platform/database && git commit -m "feat(platform): validate runtime database configuration"`

### Task 3: Add PostgreSQL authentication repositories

**Files:**
- Modify: `backend/internal/auth/domain.go`
- Modify: `backend/internal/auth/repository.go`
- Modify: `backend/internal/auth/service.go`
- Create: `backend/internal/auth/postgres_repository.go`
- Create: `backend/internal/auth/postgres_repository_test.go`

**Interfaces:**
- Consumes: shared `*gorm.DB`, existing `users`, `user_credentials`, `user_sessions`, `workshop_members` and `login_attempts` tables.
- Produces: `NewPostgresRepository(db *gorm.DB, clock func() time.Time) (*PostgresRepository, error)` implementing credential lookup, session lookup/revocation and active-membership resolution.
- Produces: `ActiveMembership { WorkshopID uuid.UUID; Role string }` and `MembershipResolver.ResolveActive(context.Context, uuid.UUID) (ActiveMembership, error)`.

- [ ] **Step 1: Write failing repository contract tests**

Add tests for normalized CITEXT lookup, inactive users, no credential, active-session lookup, expired/revoked sessions, exactly one active membership, zero memberships and multiple memberships. Assert driver no-row results map to existing domain errors while operational errors remain distinguishable.

- [ ] **Step 2: Run the repository tests and observe failure**

Run: `cd backend; go test ./internal/auth -run 'TestPostgresRepository' -v`
Expected: FAIL because `PostgresRepository` and the richer membership contract do not exist.

- [ ] **Step 3: Implement the repository with GORM/raw SQL**

Use explicit column lists and `resolve_active_memberships(?)`; never use `SELECT *` or `AutoMigrate`. Preserve `[32]byte` token digests as `bytea`.

- [ ] **Step 4: Update service fakes for `ActiveMembership`**

Modify `backend/internal/auth/service_test.go` helpers so all existing auth tests compile against the new resolver signature.

- [ ] **Step 5: Run auth tests**

Run: `cd backend; go test ./internal/auth`
Expected: PASS.

- [ ] **Step 6: Commit persistence adapters**

Run: `git add backend/internal/auth && git commit -m "feat(auth): add PostgreSQL runtime repositories"`

### Task 4: Make login and password rotation transaction-safe

**Files:**
- Modify: `backend/internal/auth/domain.go`
- Modify: `backend/internal/auth/repository.go`
- Modify: `backend/internal/auth/session.go`
- Modify: `backend/internal/auth/session_test.go`
- Modify: `backend/internal/auth/service.go`
- Modify: `backend/internal/auth/service_test.go`
- Modify: `backend/internal/auth/postgres_repository.go`
- Modify: `backend/internal/auth/postgres_repository_test.go`

**Interfaces:**
- Consumes: credential hash verified by `PasswordHasher`, generated `NewSession` and shared PostgreSQL transaction.
- Produces: `SessionRepository.InsertForCredential(context.Context, string, NewSession) (Session, error)` where the string is the verified hash.
- Produces: `SessionRepository.ChangePasswordAndInsert(context.Context, uuid.UUID, string, string, time.Time, NewSession) (Session, error)` with expected hash then replacement hash.
- Produces: `ErrCredentialChanged` mapped to a generic authentication/password error.

- [ ] **Step 1: Write failing race and rollback tests**

Add `TestLoginRejectsCredentialChangedAfterVerification`, `TestPasswordChangeRevokesAndCreatesSessionAtomically`, `TestLoginSessionRollbackPreservesPreviousSessions` and `TestPasswordRotationRollbackPreservesOldCredential`. Assert the new session cannot survive when the expected hash no longer matches.

- [ ] **Step 2: Run focused tests and observe failure**

Run: `cd backend; go test ./internal/auth -run 'CredentialChanged|Atomically|Rollback' -v`
Expected: FAIL because atomic repository operations do not exist.

- [ ] **Step 3: Implement session preparation and atomic operations**

Generate raw token/digests before the transaction. Inside the transaction lock `user_credentials` with `FOR UPDATE`, compare the stored hash, revoke active sessions, update the credential when required and insert exactly one new session.

- [ ] **Step 4: Route login and password change through atomic operations**

Remove the separate revoke-then-create sequence from `AuthService`. Keep the external handler contract unchanged.

- [ ] **Step 5: Run auth tests including the race detector**

Run: `cd backend; go test -race ./internal/auth`
Expected: PASS.

- [ ] **Step 6: Commit transaction-safe authentication**

Run: `git add backend/internal/auth && git commit -m "fix(auth): serialize login and password rotation"`

### Task 5: Enforce non-bypassable login throttling and trusted client IPs

**Files:**
- Modify: `backend/internal/auth/service.go`
- Modify: `backend/internal/auth/service_test.go`
- Modify: `backend/internal/auth/handler.go`
- Modify: `backend/internal/auth/handler_test.go`
- Modify: `backend/internal/auth/postgres_repository.go`
- Modify: `backend/internal/auth/postgres_repository_test.go`
- Modify: `backend/platform/security/ratelimit.go`
- Modify: `backend/platform/security/ratelimit_test.go`

**Interfaces:**
- Consumes: normalized email, trusted client IP and 15-minute failure window.
- Produces: `LoginLimiter.Allow(context.Context, string, string) (bool, error)` enforcing both IP-only and email-plus-IP thresholds at five failures.
- Produces: `LoginLimiter.RecordFailure(context.Context, string, string) error` persisted in `login_attempts`.
- Produces: `requestIP(*gin.Context) string`, relying on Gin trusted proxies configured by Task 7.

- [ ] **Step 1: Write failing bypass and proxy tests**

Add `TestLimiterBlocksRotatingEmailsFromOneIP`, `TestLimiterSeparatesIndependentIPs`, `TestLoginReturns429BeforePasswordVerification` and `TestRequestIPIgnoresUntrustedForwardedFor`. Count hasher calls to prove throttled requests do no Argon2 work.

- [ ] **Step 2: Run focused tests and observe failure**

Run: `cd backend; go test ./internal/auth ./platform/security -run 'Limiter|429|RequestIP' -v`
Expected: FAIL because only the email-plus-IP bucket exists.

- [ ] **Step 3: Implement dual persisted buckets**

Query failures since `clock().Add(-15*time.Minute)` for the IP-only and normalized-email-plus-IP buckets. Return fail-closed on repository errors and keep `Retry-After: 900`.

- [ ] **Step 4: Run auth and security tests**

Run: `cd backend; go test -race ./internal/auth ./platform/security`
Expected: PASS.

- [ ] **Step 5: Commit rate-limit hardening**

Run: `git add backend/internal/auth backend/platform/security && git commit -m "fix(auth): prevent login throttle bypass"`

### Task 6: Unify principal middleware and workshop routes on Gin

**Files:**
- Modify: `backend/platform/httpx/principal.go`
- Modify: `backend/platform/httpx/principal_test.go`
- Modify: `backend/internal/auth/service.go`
- Modify: `backend/internal/auth/handler.go`
- Modify: `backend/internal/auth/handler_test.go`
- Modify: `backend/internal/workshops/routes.go`
- Modify: `backend/internal/workshops/routes_test.go`

**Interfaces:**
- Consumes: `ActiveMembership`, credential identity and restored session.
- Produces: `AuthSession.Principal httpx.Principal` with user, workshop, role and password-change state.
- Produces: `Handler.RequireSession()` storing `httpx.Principal` in Gin context.
- Produces: `httpx.PrincipalFromGin(*gin.Context) (Principal, error)` and `httpx.RequireRoles(...string) gin.HandlerFunc`.
- Produces: `workshops.RegisterRoutes(gin.IRouter, *Handler, gin.HandlerFunc)` registering protected `/api/v1/me` and `/api/v1/workshops/current`.

- [ ] **Step 1: Write failing principal and route tests**

Add tests for restored principal fields, missing principal `401`, forced-password gate `403`, invalid role `403`, active OWNER responses and cross-tenant/missing membership `403`. Assert API errors use the common JSON envelope and request ID.

- [ ] **Step 2: Run focused tests and observe failure**

Run: `cd backend; go test ./platform/httpx ./internal/auth ./internal/workshops -run 'Principal|Me|Current|PasswordChange|Role' -v`
Expected: FAIL because workshops still use `net/http` context.

- [ ] **Step 3: Adapt principal and workshop transport to Gin**

Keep `workshops.Service.Resolve` tenant-scoped. Do not accept workshop or role from headers, query strings or request bodies.

- [ ] **Step 4: Log panic request IDs without panic values or headers**

Update `httpx.Recovery` so the server log contains only a generic panic marker and the existing request ID. Extend the existing secret-leak test.

- [ ] **Step 5: Run transport and service tests**

Run: `cd backend; go test -race ./platform/httpx ./internal/auth ./internal/workshops`
Expected: PASS.

- [ ] **Step 6: Commit unified routing**

Run: `git add backend/platform/httpx backend/internal/auth backend/internal/workshops && git commit -m "feat(api): compose authenticated workshop routes"`

### Task 7: Wire the real API and deliver bootstrap in containers

**Files:**
- Modify: `backend/cmd/api/main.go`
- Create: `backend/cmd/api/app.go`
- Create: `backend/cmd/api/app_test.go`
- Modify: `backend/Dockerfile`
- Modify: `compose.yaml`
- Modify: `compose.production.yaml`
- Modify: `.env.example`
- Modify: `infra/postgres/init-roles.sh`

**Interfaces:**
- Consumes: Tasks 2-6 constructors and fixed database role names.
- Produces: `buildApplication(config.Config) (*application, error)` containing the Gin handler, DB close function and readiness ping.
- Produces: backend image paths `/api` and `/bootstrap`.
- Produces: Compose `bootstrap` service using `TF_BOOTSTRAP_DATABASE_URL` and `entrypoint: ["/bootstrap"]`.

- [ ] **Step 1: Write failing composition tests**

Add `TestBuildApplicationRegistersAuthAndWorkshopRoutes`, `TestBuildApplicationReadinessUsesDatabase`, `TestBuildApplicationRejectsInvalidTrustedProxy` and `TestBuildApplicationClosesDatabase`. Use injectable constructors, not a live database.

- [ ] **Step 2: Run the API tests and observe failure**

Run: `cd backend; go test ./cmd/api -v`
Expected: FAIL because the composition root still passes empty dependencies.

- [ ] **Step 3: Implement the composition root**

Set Gin trusted proxies from validated config, construct all adapters/handlers, register auth plus protected workshop routes and use a 10-second graceful shutdown path.

- [ ] **Step 4: Build both container binaries**

Update the build stage to emit `/out/api` and `/out/bootstrap`; copy both to the distroless image while retaining `/api` as the default entrypoint.

- [ ] **Step 5: Standardize role names and bootstrap invocation**

Remove `DB_*_USER` overrides from init and Compose, retain password variables, add `TF_ALLOWED_ORIGIN` and `TF_TRUSTED_PROXIES`, and make bootstrap accept its password only from stdin.

- [ ] **Step 6: Verify build and Compose shape**

Run: `cd backend; go test ./cmd/api; go build ./cmd/api ./cmd/bootstrap; cd ..; docker compose config --quiet; docker build -t tallerflow-backend:sprint1 backend`
Expected: all commands pass and `docker run --rm --entrypoint /bootstrap tallerflow-backend:sprint1 -h` reaches the bootstrap binary without embedding a password.

- [ ] **Step 7: Commit runtime/container assembly**

Run: `git add backend/cmd/api backend/Dockerfile compose.yaml compose.production.yaml .env.example infra/postgres/init-roles.sh && git commit -m "feat(runtime): wire API and bootstrap containers"`

### Task 8: Align frontend contracts and accessibility

**Files:**
- Modify: `frontend/src/modules/session/types.ts`
- Modify: `frontend/src/modules/session/api.ts`
- Modify: `frontend/src/modules/session/api.test.ts`
- Modify: `frontend/src/modules/session/session.store.ts`
- Modify: `frontend/src/modules/session/session.store.test.ts`
- Modify: `frontend/src/modules/auth/LoginView.vue`
- Modify: `frontend/src/modules/auth/LoginView.test.ts`
- Modify: `frontend/src/modules/auth/ChangePasswordView.vue`
- Create: `frontend/src/modules/auth/ChangePasswordView.test.ts`
- Modify: `frontend/src/components/UiAlert.vue`
- Modify: `frontend/src/styles/main.css`

**Interfaces:**
- Consumes: backend JSON envelopes from Task 6.
- Produces: exact TypeScript DTOs for `AuthSessionPayload`, `ApiPrincipal`, `WorkshopAccess` and API error envelope.
- Produces: in-memory-only CSRF state and router states `anonymous`, `password-change-required` and `authenticated`.

- [ ] **Step 1: Write failing contract and state tests**

Assert session restoration maps the backend envelope, `401` clears state, forced change never requests workshop endpoints, a successful change rotates CSRF state, logout sends CSRF and no auth value is written to `localStorage` or `sessionStorage`.

- [ ] **Step 2: Write failing keyboard and error-announcement tests**

Assert labels connect to inputs, Enter submits, loading disables duplicate submit, server errors use an `aria-live` alert and password policy text is associated with the new-password input.

- [ ] **Step 3: Run frontend tests and observe failure**

Run: `cd frontend; npm test`
Expected: FAIL on the backend envelope/accessibility mismatches.

- [ ] **Step 4: Implement contract and accessibility corrections**

Keep credentials/cookies out of JavaScript storage. Preserve the Spanish generic login error and responsive 360 px layout.

- [ ] **Step 5: Run tests and build**

Run: `cd frontend; npm test; npm run build`
Expected: PASS.

- [ ] **Step 6: Commit frontend integration**

Run: `git add frontend && git commit -m "fix(frontend): align authenticated session flow"`

### Task 9: Add real database and browser E2E coverage to CI

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`
- Create: `frontend/playwright.config.ts`
- Create: `frontend/e2e/auth-flow.spec.ts`
- Create: `backend/internal/auth/postgres_integration_test.go`
- Create: `scripts/verify-sprint1.ps1`
- Create: `scripts/verify-sprint1.sh`
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: Compose services, bootstrap stdin command, API at the gateway and Vue routes.
- Produces: repeatable local/CI verification scripts and Playwright's Chromium suite.

- [ ] **Step 1: Add failing live PostgreSQL integration tests**

Gate tests on `TF_TEST_DATABASE_URL`. Cover credential/session persistence, concurrent old-password login rejection, dual rate-limit buckets and tenant context isolation against migrated PostgreSQL.

- [ ] **Step 2: Add failing Playwright user-journey tests**

Cover invalid login, forced password change, `/me`, current workshop, logout, expired/invalid session, CSRF rejection, `429`, keyboard submit and a 360x800 viewport. Assert secrets and tokens never appear in browser storage.

- [ ] **Step 3: Add cross-platform Sprint verification scripts**

Both scripts must run Go unit/race tests, frontend tests/build, clean Compose startup, Flyway validation, SQL tests, bootstrap once/again, Playwright and teardown. They must stop on first failure and never echo supplied secrets.

- [ ] **Step 4: Run the new suites and observe initial failure**

Run: `powershell -ExecutionPolicy Bypass -File scripts/verify-sprint1.ps1`
Expected: initial failures identify remaining live integration gaps; fix only code owned by Tasks 2-8 until the script passes.

- [ ] **Step 5: Update CI to mirror the verified script**

Add backend race tests, live PostgreSQL integration tests, both-image checks, Playwright Chromium and a Compose smoke job. Preserve failure logs and unconditional volume teardown without printing environment secrets.

- [ ] **Step 6: Run the complete verification script from empty volumes**

Run: `powershell -ExecutionPolicy Bypass -File scripts/verify-sprint1.ps1`
Expected: PASS, including first bootstrap success and second bootstrap rejection.

- [ ] **Step 7: Commit E2E and CI**

Run: `git add frontend/package.json frontend/package-lock.json frontend/playwright.config.ts frontend/e2e backend/internal/auth/postgres_integration_test.go scripts .github/workflows/ci.yml && git commit -m "test: verify Sprint 1 end to end"`

### Task 10: Document verified operation and land Sprint 1

**Files:**
- Modify: `README.md`
- Modify: `docs/sprints/sprint-01-human-handoff.md`
- Modify: `docs/sprints/sprint-01-team-plan.md`
- Modify: `docs/contracts/openapi.yaml`

**Interfaces:**
- Consumes: commands and HTTP contracts proven by Task 9.
- Produces: accurate runbook, checked Sprint acceptance list and final OpenAPI contract.

- [ ] **Step 1: Update documentation only from passing evidence**

Document clean startup, stdin bootstrap, development URLs, verification scripts and teardown. Mark a checklist item complete only when the corresponding Task 9 command passed.

- [ ] **Step 2: Validate OpenAPI against implemented DTOs and routes**

Confirm login/session/change-password, `/me`, current workshop, error envelopes, `401`, `403` and `429` match Tasks 6 and 8.

- [ ] **Step 3: Run formatting, diff and secret checks**

Run: `cd backend; gofmt -w cmd internal platform; go vet ./...; cd ..; git diff --check; git grep -n -I -E '(BEGIN (RSA|OPENSSH|EC) PRIVATE KEY|postgres://[^:]+:[^@]+@|TF_SESSION_PEPPER=.+)' -- ':!docs/superpowers/plans/*'`
Expected: formatter/vet/diff pass and grep returns no real credentials.

- [ ] **Step 4: Run the final clean verification twice**

Run: `powershell -ExecutionPolicy Bypass -File scripts/verify-sprint1.ps1` twice, with teardown between runs.
Expected: both runs PASS, proving reproducibility.

- [ ] **Step 5: Commit verified documentation**

Run: `git add README.md docs/sprints docs/contracts/openapi.yaml && git commit -m "docs: complete Sprint 1 runbook"`

- [ ] **Step 6: Review the complete branch against local main**

Run: `git diff --check main...HEAD; git log --oneline --decorate main..HEAD; git status --short`
Expected: no diff errors, expected contributor/implementation commits and a clean worktree.

- [ ] **Step 7: Push the reviewed integration branch**

Run: `git push -u origin codex/s1-sprint1-final`
Expected: fast-forward/new-branch push without force.

- [ ] **Step 8: Fast-forward local main and retest**

From the root checkout, preserve its untracked files and run `git merge --ff-only codex/s1-sprint1-final`, followed by the complete verification script.
Expected: `main` points at the reviewed Sprint 1 result and untracked root files are unchanged.

- [ ] **Step 9: Publish main**

Run: `git push origin main`
Expected: `origin/main` advances without force and contains all four Sprint 1 contributions plus the verified integration fixes.
