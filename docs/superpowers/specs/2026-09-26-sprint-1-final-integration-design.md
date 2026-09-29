# TallerFlow Sprint 1 Final Integration Design

## Outcome

Sprint 1 ends with one reproducible application in `main`: PostgreSQL and
Flyway start from empty storage, bootstrap creates exactly one OWNER, that
OWNER completes the forced password change and reaches the current workshop,
and logout, CSRF, rate limiting and tenant isolation work through the real
Vue-to-Go-to-PostgreSQL path.

The final result must satisfy the Sprint 1 definition of done in
`docs/sprints/sprint-01-team-plan.md`, including clean Compose startup,
backend/frontend/data/E2E checks, the 360 px login experience and documented
commands that were actually executed.

## Source Branches and Integration Strategy

The integration branch is `codex/s1-sprint1-final`. It starts from
`codex/s1-michael-reconcile`, which already contains:

- Stefano's data and infrastructure work through `4adf889`.
- Jose's API, authentication and bootstrap work through `1ce90d6`.
- Michael's tenancy work reconciled with the final persistence contract in
  `27363a8`.

Lucero's completed branch, `origin/feat/s1-lucero-frontend-shell` at
`96af725`, is merged next. The original Michael branch remains preserved on
the remote; `27363a8` is the accepted adaptation because it changes
`memberships` to `workshop_members` and aligns `status` and `timezone` with
the final Flyway schema.

No work is merged to `main` until this integration branch passes every
acceptance gate. The uncommitted files in the root checkout and Jose's dirty
worktree are not part of this integration and must remain untouched.

## Runtime Architecture

### Composition Root

`backend/cmd/api/main.go` owns process composition. It will:

1. Load and validate configuration.
2. Open the application database connection and verify it.
3. Construct the authentication repositories, session service, membership
   resolver, rate limiter and auth handler.
4. Construct Michael's tenant runner and workshop handler.
5. Register health, authentication, `/api/v1/me` and
   `/api/v1/workshops/current` on one Gin router.
6. Close database resources during graceful shutdown.

The router remains the only global HTTP assembly point. Feature packages
expose route registration functions and do not create global routers.
Michael's existing `net/http` handlers will be adapted to Gin so the
application has one middleware, context and error model.

### Authentication Persistence

Production adapters will implement the existing credential, session,
membership and login-limiter contracts against PostgreSQL through the shared
runtime `*gorm.DB`. Readiness uses the underlying `database/sql` connection's
`PingContext`; the bootstrap command keeps its separate pgx connection and
restricted role. Runtime adapters use the application role and the existing
schema; Flyway remains the only schema owner. Raw passwords, session tokens,
CSRF tokens and connection strings are never logged or stored in clear text.

Login resolves exactly one active workshop membership and returns a complete
authenticated principal: user ID, email, display name, workshop ID, role and
the forced-password-change flag. Session restoration rebuilds the same
principal, so downstream handlers never infer tenant identity from request
parameters.

### Tenant Boundary

Every tenant-owned query runs through `GormTenantRunner.WithinTenant`, which
sets `app.workshop_id` transaction-locally before executing work. `/me` reads
the authenticated principal. `/workshops/current` verifies active membership
and reads the workshop inside the tenant transaction. Missing session returns
`401`; authenticated users without access return `403`; another workshop is
never disclosed.

### Session and Password-Change Race

Successful login is finalized transactionally. After Argon2 verification,
the repository locks the credential row, confirms that the stored password
hash is still the hash that was verified, revokes previous sessions and
inserts the replacement session in one transaction. If a concurrent password
change updated the hash, login fails and cannot create a post-revocation
session from the old password.

Password change locks the credential row, verifies the current credential,
updates the password hash, clears `must_change_password`, revokes all existing
sessions and creates the replacement session atomically.

### Rate Limiting

The PostgreSQL login-attempt repository enforces two limits before Argon2
work:

- A per-client-IP bucket prevents attackers from rotating email addresses to
  force unbounded password hashing.
- A normalized-email-plus-IP bucket slows targeted guessing without globally
  locking an account.

The backend accepts forwarded client addresses only from the internal Caddy
proxy path; direct untrusted forwarding headers are ignored. Exceeded limits
return the documented `429` envelope and `Retry-After` value.

### Database Roles

Sprint 1 standardizes the PostgreSQL role names used by migrations:
`tallerflow_migrator`, `tallerflow_app` and `tallerflow_bootstrap`. Role names
are not configurable; their passwords and connection URLs are. This removes
the mismatch between configurable initialization names and hard-coded Flyway
grants while keeping deployment secrets configurable.

### Bootstrap Delivery

The backend image contains both `/api` and `/bootstrap` binaries. Compose
provides a one-shot bootstrap tool that uses the separate bootstrap database
credential and accepts the initial password only through standard input. It
does not place the password in Compose YAML, environment variables, process
arguments or logs. A second execution returns `BOOTSTRAP_ALREADY_EXISTS` and
does not create partial identity data.

## Frontend Integration

Lucero's Vue application remains the presentation layer. Its session store
continues to keep the opaque session in an HttpOnly cookie and the CSRF value
in memory only; it must not use `localStorage` or `sessionStorage` for
credentials or tokens.

The HTTP contracts are aligned with OpenAPI and the backend envelope:

- Login and password change consume `data.csrf_token` and
  `data.must_change_password`.
- `/api/v1/me` returns the principal fields expected by the store.
- `/api/v1/workshops/current` returns the active workshop and role.
- Authentication failures retain a generic user-facing message.

Router guards preserve the forced-password-change flow. The login and change
password forms remain keyboard-operable, visibly label validation errors and
work at a 360 px viewport.

## Error Handling and Observability

API errors use one JSON envelope with a request ID. Expected authentication,
authorization, CSRF, validation and throttling failures never expose database
or cryptographic details. Unexpected failures are logged server-side with the
request ID but without cookies, CSRF values, passwords, hashes or database
URLs. Readiness reports success only after the real database ping succeeds.

Startup fails fast when required configuration, database connectivity or
handler composition is invalid.

## Verification and CI

The branch is eligible for `main` only after all of the following pass:

1. `go test ./...` and `go test -race ./...`.
2. Frontend unit tests and production build.
3. Flyway migration and SQL contract, constraint and RLS tests against a
   fresh PostgreSQL volume.
4. Backend container build containing both API and bootstrap binaries.
5. Clean Compose startup with healthy PostgreSQL, successful Flyway and ready
   API.
6. Bootstrap succeeds once and the second attempt is rejected.
7. E2E covers invalid login, forced password change, successful login,
   `/me`, current workshop, logout, expired session, CSRF rejection and rate
   limiting.
8. Tenant tests prove missing context and a different workshop cannot read
   data.
9. Frontend accessibility checks cover keyboard login and the 360 px layout.
10. Secret scanning confirms no real secret or raw token is tracked or logged.

CI will run the same backend, frontend and database commands and include a
container-level smoke test. README and runbooks are updated only with commands
that passed locally.

## Merge and Publication

After all checks pass, `codex/s1-sprint1-final` is pushed for final review.
Local `main` is updated without discarding its six documentation commits,
then the verified integration branch is merged without rewriting contributor
history. The resulting `main` is tested once more and pushed to `origin/main`.

No force push, history rewrite or cleanup of contributor branches is part of
this Sprint 1 completion.

## Explicit Non-Goals

- Customers, orders, production stages, tracking and image upload.
- Domain, VPS and external backup provider configuration.
- Multi-workshop switching for one user; Sprint 1 requires exactly one active
  membership at login.
- Replacing Flyway or introducing a second schema-management mechanism.
