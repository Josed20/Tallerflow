# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

TallerFlow is a multi-tenant traceability platform for garment workshops (talleres de confección). Go API + Vue 3 SPA + PostgreSQL with row-level security, all run through Docker Compose. README, docs and UI copy are in Spanish; product context lives in `docs/contexto-proyecto.md`, API contracts in `docs/contracts/` (OpenAPI), and sprint plans in `docs/superpowers/plans/`.

## Commands

Toolchain: Go 1.27.x, Node 24, Docker Compose v2. PostgreSQL and Flyway only run in containers.

```powershell
# Local environment
Copy-Item .env.example .env
docker compose up -d --wait postgres flyway            # DB + migrations only
docker compose -f compose.yaml -f compose.mail.yaml --profile app up -d --build --wait   # full app at http://localhost:8080 (+ Mailpit)
docker compose -f compose.yaml -f compose.mail.yaml --profile app --profile tools down --remove-orphans   # stop (add --volumes to wipe DB)

# Backend (module github.com/Josed20/Tallerflow/backend)
go -C backend test ./...
go -C backend test ./internal/orders -run TestName -v   # single test
go -C backend vet ./...

# Frontend
npm --prefix frontend run dev                            # Vite HMR at http://localhost:5173, proxies /api + /health to the Compose stack on :8080
npm --prefix frontend test                               # vitest (src/**/*.test.ts)
npm --prefix frontend exec vitest run src/modules/orders/OrdersListView.test.ts   # single file
npm --prefix frontend run build                          # vue-tsc type check + vite build
npm --prefix frontend run test:e2e -- onboarding-flow.spec.ts   # Playwright, needs running app; E2E_BASE_URL overrides http://127.0.0.1:8080
```

Postgres integration tests (`*_integration_test.go`, `TestPostgresIntegration*`) skip unless `TF_TEST_DATABASE_URL` (and for some, `TF_TEST_BOOTSTRAP_DATABASE_URL` / `TF_TEST_ADMIN_DATABASE_URL`) point at a migrated DB; see `scripts/verify-sprint2.sh` for the exact values.

Full verification (what CI runs, from empty isolated Compose volumes): `./scripts/verify-sprint2.sh` (or `scripts\verify-sprint2.ps1` on Windows). It runs the Sprint 1 suite first, then `go test -race`, `go vet`, vitest + build, Redocly OpenAPI lint, SQL contract tests in `database/tests/`, Postgres integration tests and Playwright E2E. It uses isolated project names/ports (`tallerflow_sprint2_verify`, PG on 55434, app on 18081) so it does not touch the normal dev volume.

## Architecture

**Request path:** Caddy (`infra/caddy/Caddyfile`) serves everything from one origin: `/api/*` and `/health/*` → Go backend, everything else → nginx serving the Vue build. Frontend fetches use relative paths with `credentials: 'include'`. In dev, `frontend/vite.config.ts` proxies to that same origin (`VITE_API_TARGET`, default `http://localhost:8080`) and rewrites `Origin`/`Referer`, because the backend rejects mutations whose origin isn't `TF_ALLOWED_ORIGIN`. `frontend/nginx.conf` serves `index.html` as `no-cache` and hashed `/assets/` as immutable.

**Backend (`backend/`)** — Gin + GORM (pgx). Layout:

- `cmd/api/app.go` is the composition root: it builds every repository/service/handler by hand and registers modules through `httpx.RouteRegistrar` funcs. New modules must be wired here.
- `cmd/bootstrap` is a CLI that creates the single initial OWNER via the bootstrap DB role (password from stdin).
- `internal/<module>/` (auth, onboarding, passwordreset, team, workshops, clients, orders) each follow handler → service → repository, with `routes.go`/`RegisterRoutes(routes, handler, requireSession)`. Private routes take `authHandler.RequireSession()`, and handlers read the caller via `httpx.PrincipalFromGin(c)` (user, workshop, role).
- `platform/` holds shared infra: `config` (all `TF_*` env vars), `database`, `httpx` (router, error envelope via `RespondError`, request IDs, principal), `security` (CSRF, rate limiting).

**Multi-tenancy is enforced in PostgreSQL via RLS.** All tenant-owned data must go through `database.TenantRunner.WithinTenant(ctx, workshopID, fn)` (`platform/database/tenant.go`), which opens a transaction and sets `app.workshop_id` transaction-locally. Without that context, protected tables return no rows. Never query tenant tables on the raw `*gorm.DB`. The exception is public-token flows (team invitation consume, customer order tracking in `internal/orders/tracking.go`): they set a transaction-local `app.<x>_token_hash` that an RLS policy matches against the stored SHA/HMAC of the token to discover the workshop, then set `app.workshop_id` in the same transaction.

**DB roles** (created by `infra/postgres/init-roles.sh`): `tallerflow_migrator` (Flyway), `tallerflow_app` (API, limited grants + RLS), `tallerflow_bootstrap` (onboarding/first-OWNER creation, separate connection `TF_BOOTSTRAP_DATABASE_URL`), `tallerflow_owner` (DB ownership only). Grants for new tables must be added in migrations.

**Migrations:** Flyway, `database/migrations/V<n>__description.sql`. Never edit an applied migration; add a new version. The backend does not use `AutoMigrate`. SQL contract/RLS tests live in `database/tests/` and are run with `psql` as the appropriate role (see README).

**Auth/session:** cookie sessions (peppered with `TF_SESSION_PEPPER`), CSRF token required on mutating requests (`X-CSRF-Token`), origin check against `TF_ALLOWED_ORIGIN`, in-memory rate limiters. Roles: OWNER, ADMIN, SUPERVISOR, OPERATOR (`internal/workshops/role.go`); a first login after CLI bootstrap forces a password change. Error responses use a `{ "error": { "code", "request_id", ... } }` envelope with stable codes.

**Frontend (`frontend/`)** — Vue 3 + TypeScript, Pinia, TanStack Vue Query, vue-router, Tailwind v4, reka-ui. Feature code lives in `src/modules/<feature>/` (view + `api.ts` + store + colocated `*.test.ts`). The shared fetch wrapper and `ApiError` are in `src/modules/session/api.ts`; role-based UI gating is in `src/modules/session/role-policy.ts` and `src/app/router.ts`. Customers have no accounts: they follow one order through a public read-only page (`modules/tracking`, route `/seguimiento/:token`) outside the internal role-aware shell (`modules/session/AppShell.vue`, used for `/app/*`). E2E tests run in Chromium at 360px width (mobile-first).

## Skill routing

When the user's request matches an available skill, invoke it via the Skill tool. When in doubt, invoke the skill.

Key routing rules:
- Product ideas/brainstorming → invoke /office-hours
- Strategy/scope → invoke /plan-ceo-review
- Architecture → invoke /plan-eng-review
- Design system/plan review → invoke /design-consultation or /plan-design-review
- Full review pipeline → invoke /autoplan
- Bugs/errors → invoke /investigate
- QA/testing site behavior → invoke /qa or /qa-only
- Code review/diff check → invoke /review
- Visual polish → invoke /design-review
- Ship/deploy/PR → invoke /ship or /land-and-deploy
- Save progress → invoke /context-save
- Resume context → invoke /context-restore
- Author a backlog-ready spec/issue → invoke /spec

## gstack (recommended)

This project uses [gstack](https://github.com/garrytan/gstack) for AI-assisted workflows.
Install it for the best experience:

```bash
git clone --depth 1 https://github.com/garrytan/gstack.git ~/.claude/skills/gstack
cd ~/.claude/skills/gstack && ./setup --team
```

Skills like /qa, /ship, /review, /investigate, and /browse become available after install.
Use /browse for all web browsing (Aside first, the bundled gstack browser as fallback). Use ~/.claude/skills/gstack/... for gstack file paths.
