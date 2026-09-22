# TallerFlow Phase 1 Foundations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Entregar una base local integrada donde un OWNER provisionado por bootstrap pueda iniciar y cerrar sesión, consultar su taller y quedar aislado de otros talleres.

**Architecture:** Vue se sirve como SPA y consume una API Go del mismo origen. Go administra contraseñas Argon2id, sesiones opacas, CSRF y permisos; PostgreSQL aplica Flyway y RLS como segunda barrera de aislamiento.

**Tech Stack:** Go 1.27.x, Gin, GORM, PostgreSQL 18.x, Flyway 13.7.0, Vue 3, TypeScript, Vite, TanStack Query, Pinia, Tailwind CSS, Reka UI, Vitest, Playwright, Docker Compose v2 y Caddy 2.x.

**Spec:** `docs/superpowers/specs/2026-09-21-tallerflow-mvp-design.md`

## Global Constraints

- No usar Supabase, JWT de navegador, `localStorage` para sesión ni `GORM AutoMigrate`.
- Flyway es la única autoridad del esquema y usa credenciales distintas de la API.
- Toda tabla privada incluye `workshop_id`; el rol de aplicación no posee tablas ni tiene `BYPASSRLS`.
- Producción será same-origin; desarrollo usa proxy de Vite hacia `http://backend:8080`.
- Cookie: `__Host-tallerflow_session`, `Secure`, `HttpOnly`, `SameSite=Lax`, `Path=/`.
- Contraseñas: Argon2id, memoria 64 MiB, 3 iteraciones, paralelismo 2, salt de 16 bytes y clave de 32 bytes.
- Tokens de sesión y recuperación: 32 bytes de `crypto/rand`; PostgreSQL guarda SHA-256, nunca el valor original.
- Timestamps en UTC; taller inicial en `America/Lima`.
- Backend: `go test -race ./...` y `go vet ./...`.
- Frontend: `npm run type-check`, `npm run lint`, `npm run test:unit -- --run` y `npm run build`.

## Review Focus

- Configuración incompleta: la API debe fallar al iniciar sin imprimir secretos; Task 1 incluye la prueba.
- Sesión expirada o revocada: debe devolver `401 SESSION_INVALID` y borrar la cookie; Task 5 incluye la prueba.
- Mutación sin CSRF u origen permitido: debe devolver `403 CSRF_INVALID`; Task 5 incluye la prueba.
- Transacción sin taller o con taller ajeno: PostgreSQL debe denegar filas; Task 3 incluye la prueba.
- Intentos repetidos de login, incluso para email inexistente: deben producir respuesta genérica y `429` sin filtrar cuentas; Task 5 incluye la prueba.

---

## Mapa de archivos

```text
backend/
├── cmd/api/main.go
├── cmd/bootstrap/main.go
├── internal/auth/{domain.go,password.go,session.go,repository.go,service.go,handler.go,routes.go}
├── internal/workshops/{domain.go,repository.go,service.go,handler.go,routes.go}
├── platform/config/config.go
├── platform/database/{open.go,tenant.go}
├── platform/httpx/{errors.go,middleware.go,router.go}
└── platform/security/{random.go,csrf.go,ratelimit.go}

frontend/src/
├── app/{main.ts,router.ts,query.ts}
├── modules/auth/{api.ts,queries.ts,LoginView.vue}
├── modules/session/{api.ts,store.ts,types.ts}
├── components/ui/
├── layouts/AuthLayout.vue
└── styles/{tokens.css,main.css}

database/migrations/
├── V1__create_extensions.sql
└── V2__create_identity_sessions_and_workshops.sql

infra/{caddy/Caddyfile,docker/}
compose.yaml
.env.example
.github/workflows/ci.yml
```

## Preflight obligatorio

Antes de crear ramas, José y Stefano verifican:

```powershell
go version
node --version
npm --version
docker version
docker compose version
git status --short
```

Expected:

- Go 1.27.x.
- Node 24 LTS.
- Docker Engine accesible y Compose v2.
- `main` sin cambios rastreados pendientes.

El entorno revisado antes de este plan no tenía Go disponible y el daemon de Docker estaba apagado. La ejecución se detiene en preflight hasta corregir ambos requisitos; la documentación puede revisarse sin ellos.

### Task 1: Backend skeleton, configuration and health

**Owner:** José  
**Branch:** `feat/s1-jose-auth-api`

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/api/main.go`
- Create: `backend/platform/config/config.go`
- Create: `backend/platform/config/config_test.go`
- Create: `backend/platform/httpx/errors.go`
- Create: `backend/platform/httpx/middleware.go`
- Create: `backend/platform/httpx/router.go`
- Create: `backend/platform/httpx/router_test.go`
- Create: `docs/contracts/openapi.yaml`

**Interfaces:**
- Produces: `config.Load() (config.Config, error)`
- Produces: `httpx.NewRouter(httpx.Dependencies) *gin.Engine`
- Produces: `GET /health/live`

- [ ] **Step 1: Inicializar el módulo y dependencias base**

```powershell
cd backend
go mod init github.com/tallerflow/tallerflow/backend
go get github.com/gin-gonic/gin github.com/google/uuid github.com/stretchr/testify
```

- [ ] **Step 2: Escribir la prueba fallida de configuración**

```go
func TestLoadRejectsMissingSecrets(t *testing.T) {
    t.Setenv("TF_DATABASE_URL", "")
    t.Setenv("TF_SESSION_PEPPER", "")
    _, err := Load()
    require.ErrorContains(t, err, "TF_DATABASE_URL")
    require.NotContains(t, err.Error(), "password=")
}
```

Run: `cd backend; go test ./platform/config -run TestLoadRejectsMissingSecrets -v`  
Expected: FAIL porque `Load` aún no existe.

- [ ] **Step 3: Implementar configuración tipada**

```go
type Config struct {
    Environment   string
    HTTPAddress   string
    DatabaseURL   string
    SessionPepper string
    AllowedOrigin string
}

func Load() (Config, error)
```

`Load` lee exclusivamente variables `TF_*`, exige database URL y pepper, usa `:8080` por defecto y no incluye valores secretos en errores.

- [ ] **Step 4: Probar request ID y health**

```go
func TestHealthLiveReturnsRequestID(t *testing.T) {
    r := NewRouter(Dependencies{})
    req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
    res := httptest.NewRecorder()
    r.ServeHTTP(res, req)
    require.Equal(t, http.StatusOK, res.Code)
    require.NotEmpty(t, res.Header().Get("X-Request-ID"))
}
```

Implementar `RequestID` y una respuesta `{"data":{"status":"alive"}}`.

- [ ] **Step 5: Publicar el contrato inicial**

`docs/contracts/openapi.yaml` usa OpenAPI 3.1 y define exactamente `/health/live`, `/health/ready`, `/api/v1/auth/login`, `/logout`, `/session`, `/me` y `/workshops/current`. Sus schemas son `ApiError`, `LoginInput`, `SessionData`, `UserSummary`, `WorkshopSummary` y `MeData`.

```yaml
openapi: 3.1.0
info: { title: TallerFlow API, version: 0.1.0 }
paths:
  /health/live:
    get:
      operationId: healthLive
      responses:
        '200': { description: API process is alive }
```

- [ ] **Step 6: Verificar y confirmar**

```powershell
cd backend
gofmt -w .
go test -race ./...
go vet ./...
git add backend
git commit -m "feat(platform): bootstrap Go API and health endpoint"
```

### Task 2: PostgreSQL, roles and Flyway identity schema

**Owner:** Stefano  
**Branch:** `feat/s1-stefano-data-platform`

**Files:**
- Create: `database/migrations/V1__create_extensions.sql`
- Create: `database/migrations/V2__create_identity_sessions_and_workshops.sql`
- Create: `infra/docker/postgres/init/001_roles.sh`
- Create: `infra/docker/postgres/verify/schema.sql`
- Create: `.env.example`
- Create: `compose.yaml`

**Interfaces:**
- Produces: roles `tallerflow_owner`, `tallerflow_app`, `tallerflow_flyway`
- Produces: tables `users`, `user_credentials`, `user_sessions`, `login_attempts`, `password_reset_tokens`, `workshops`, `workshop_members`, `audit_events`

- [ ] **Step 1: Crear Compose mínimo con PostgreSQL y Flyway**

```yaml
services:
  postgres:
    image: postgres:18.6
    environment:
      POSTGRES_DB: tallerflow
      POSTGRES_USER: tallerflow_owner
      POSTGRES_PASSWORD: ${TF_DB_OWNER_PASSWORD}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U tallerflow_owner -d tallerflow"]
      interval: 5s
      timeout: 3s
      retries: 20
  flyway:
    image: redgate/flyway:13.7.0
    command: -connectRetries=60 migrate
    environment:
      FLYWAY_URL: jdbc:postgresql://postgres:5432/tallerflow
      FLYWAY_USER: tallerflow_flyway
      FLYWAY_PASSWORD: ${TF_DB_FLYWAY_PASSWORD}
    volumes:
      - ./database/migrations:/flyway/sql:ro
    depends_on:
      postgres:
        condition: service_healthy
```

`001_roles.sh` crea los roles usando secretos del entorno sin escribirlos en SQL versionado:

```sh
#!/bin/sh
set -eu
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  --set=app_password="$TF_DB_APP_PASSWORD" \
  --set=flyway_password="$TF_DB_FLYWAY_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE tallerflow_app LOGIN PASSWORD %L', :'app_password') \gexec
SELECT format('CREATE ROLE tallerflow_flyway LOGIN PASSWORD %L', :'flyway_password') \gexec
GRANT CONNECT ON DATABASE tallerflow TO tallerflow_app, tallerflow_flyway;
GRANT USAGE, CREATE ON SCHEMA public TO tallerflow_flyway;
SQL
```

- [ ] **Step 2: Crear extensiones y tablas**

`V1` habilita `citext`. `V2` crea UUID con `gen_random_uuid()`, checks de estado y rol, índices de expiración y claves foráneas. La tabla de sesiones usa `token_hash BYTEA UNIQUE NOT NULL` y `csrf_token_hash BYTEA NOT NULL`.

```sql
CREATE TABLE workshops (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name varchar(160) NOT NULL,
  timezone varchar(80) NOT NULL DEFAULT 'America/Lima',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE workshop_members (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workshop_id uuid NOT NULL REFERENCES workshops(id),
  user_id uuid NOT NULL REFERENCES users(id),
  role varchar(20) NOT NULL CHECK (role IN ('OWNER','ADMIN','OPERATOR')),
  status varchar(20) NOT NULL CHECK (status IN ('ACTIVE','INACTIVE')),
  UNIQUE (workshop_id, user_id)
);
```

- [ ] **Step 3: Añadir políticas RLS iniciales**

```sql
ALTER TABLE workshop_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE workshop_members FORCE ROW LEVEL SECURITY;
CREATE POLICY workshop_members_tenant ON workshop_members
USING (
  workshop_id = nullif(current_setting('app.workshop_id', true), '')::uuid
);
```

Aplicar el mismo patrón a `audit_events`. `users`, credenciales y sesiones solo se acceden mediante repositorios de autenticación con grants explícitos; no se exponen a consultas de dominio.

Crear una función `SECURITY DEFINER` con `search_path = pg_catalog, public` que reciba `user_id` y devuelva exclusivamente las membresías activas de ese usuario. Revocar ejecución pública y concederla solo a `tallerflow_app`. Michael usará esta función para resolver el taller antes de abrir la transacción tenant.

```sql
CREATE FUNCTION resolve_active_memberships(p_user_id uuid)
RETURNS TABLE (workshop_id uuid, role varchar)
LANGUAGE sql SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
  SELECT wm.workshop_id, wm.role
  FROM public.workshop_members wm
  WHERE wm.user_id = p_user_id AND wm.status = 'ACTIVE'
$$;
REVOKE ALL ON FUNCTION resolve_active_memberships(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION resolve_active_memberships(uuid) TO tallerflow_app;
```

- [ ] **Step 4: Verificar esquema desde una base vacía**

```powershell
docker compose up -d postgres
docker compose run --rm flyway validate
docker compose run --rm flyway migrate
docker compose run --rm flyway migrate
```

Expected: la última ejecución informa que el esquema está actualizado.

- [ ] **Step 5: Confirmar**

```powershell
git add compose.yaml .env.example database infra/docker/postgres
git commit -m "feat(database): add identity schema and tenant policies"
```

### Task 3: Database connection and tenant transaction

**Owner:** Michael  
**Branch:** `feat/s1-michael-tenancy-team`

**Files:**
- Create: `backend/platform/database/open.go`
- Create: `backend/platform/database/tenant.go`
- Create: `backend/platform/database/tenant_integration_test.go`

**Interfaces:**
- Consumes: migrated PostgreSQL from Task 2
- Produces: `database.Open(ctx, databaseURL) (*gorm.DB, error)`
- Produces: `TenantRunner.WithWorkshop(ctx, workshopID, fn) error`

- [ ] **Step 1: Escribir prueba de denegación sin contexto**

```go
func TestWorkshopMembersDenyWithoutTenant(t *testing.T) {
    db := integrationDB(t)
    var count int64
    err := db.Table("workshop_members").Count(&count).Error
    require.NoError(t, err)
    require.Zero(t, count)
}
```

Run: `cd backend; go test ./platform/database -run TestWorkshopMembersDenyWithoutTenant -v`  
Expected: FAIL hasta usar el rol `tallerflow_app` y activar RLS.

- [ ] **Step 2: Definir runner transaccional**

```go
type TenantRunner struct{ DB *gorm.DB }

func (r TenantRunner) WithWorkshop(
    ctx context.Context,
    workshopID uuid.UUID,
    fn func(tx *gorm.DB) error,
) error
```

Dentro de la transacción ejecutar `SELECT set_config('app.workshop_id', ?, true)` antes de `fn`. Rechazar UUID nulo.

- [ ] **Step 3: Probar aislamiento cruzado**

```go
func TestTenantRunnerCannotReadAnotherWorkshop(t *testing.T) {
    a, b := seedTwoWorkshops(t)
    err := runner.WithWorkshop(ctx, a.ID, func(tx *gorm.DB) error {
        var members []WorkshopMemberRow
        require.NoError(t, tx.Find(&members).Error)
        require.NotEmpty(t, members)
        for _, member := range members {
            require.Equal(t, a.ID, member.WorkshopID)
            require.NotEqual(t, b.ID, member.WorkshopID)
        }
        return nil
    })
    require.NoError(t, err)
}
```

- [ ] **Step 4: Probar readiness**

Agregar `database.Ping(ctx)` y conectar `GET /health/ready`; debe devolver `503 DEPENDENCY_UNAVAILABLE` si PostgreSQL no responde.

- [ ] **Step 5: Verificar y confirmar**

```powershell
cd backend
go test -race ./platform/database ./platform/httpx
git add platform cmd
git commit -m "feat(database): enforce tenant-scoped transactions"
```

### Task 4: Password hashing and session service

**Owner:** José  
**Branch:** `feat/s1-jose-auth-api`

**Files:**
- Create: `backend/internal/auth/domain.go`
- Create: `backend/internal/auth/password.go`
- Create: `backend/internal/auth/password_test.go`
- Create: `backend/internal/auth/repository.go`
- Create: `backend/internal/auth/session.go`
- Create: `backend/internal/auth/session_test.go`
- Create: `backend/platform/security/random.go`

**Interfaces:**
- Produces: `PasswordHasher.Hash(password string) (string, error)`
- Produces: `PasswordHasher.Verify(encoded, password string) (bool, error)`
- Produces: `SessionService.Create(ctx, userID, metadata) (RawSession, error)`
- Produces: `SessionService.Authenticate(ctx, rawToken) (Session, error)`
- Produces: `SessionService.Revoke(ctx, sessionID) error`

- [ ] **Step 1: Escribir pruebas Argon2id**

```go
func TestPasswordHasherRoundTrip(t *testing.T) {
    h := NewPasswordHasher(DefaultPasswordParams())
    encoded, err := h.Hash("Correct horse battery staple 7!")
    require.NoError(t, err)
    ok, err := h.Verify(encoded, "Correct horse battery staple 7!")
    require.NoError(t, err)
    require.True(t, ok)
    require.Contains(t, encoded, "$argon2id$v=19$m=65536,t=3,p=2$")
}
```

También probar contraseña incorrecta, hash malformado y límite de 1 MiB de entrada antes de calcular.

- [ ] **Step 2: Implementar hasher**

```go
type PasswordParams struct {
    Memory uint32
    Iterations uint32
    Parallelism uint8
    SaltLength uint32
    KeyLength uint32
}

func DefaultPasswordParams() PasswordParams {
    return PasswordParams{65536, 3, 2, 16, 32}
}
```

Usar `crypto/rand`, `argon2.IDKey`, formato PHC y `subtle.ConstantTimeCompare`.

- [ ] **Step 3: Escribir prueba de sesión opaca**

```go
func TestCreateStoresOnlyTokenHash(t *testing.T) {
    repo := newFakeSessionRepository()
    raw, err := service.Create(ctx, userID, SessionMetadata{})
    require.NoError(t, err)
    require.Len(t, raw.Token, 43)
    expected := security.TokenDigest([]byte("test-pepper"), raw.Token)
    require.NotEqual(t, []byte(raw.Token), repo.inserted.TokenHash[:])
    require.Equal(t, expected, repo.inserted.TokenHash)
}
```

- [ ] **Step 4: Implementar repositorio y servicio**

```go
type SessionRepository interface {
    Insert(context.Context, NewSession) (Session, error)
    FindActiveByTokenHash(context.Context, [32]byte, time.Time) (Session, error)
    Revoke(context.Context, uuid.UUID, time.Time) error
    RevokeAllForUser(context.Context, uuid.UUID, time.Time) error
}
```

`security.TokenDigest` usa HMAC-SHA-256 con el pepper del servidor. Expirar a ocho horas y rotar en cada login.

- [ ] **Step 5: Verificar y confirmar**

```powershell
cd backend
go test -race ./internal/auth ./platform/security
git add internal/auth platform/security
git commit -m "feat(auth): add Argon2id credentials and opaque sessions"
```

### Task 5: Login, cookies, CSRF and throttling

**Owner:** José  
**Branch:** `feat/s1-jose-auth-api`

**Files:**
- Create: `backend/internal/auth/service.go`
- Create: `backend/internal/auth/service_test.go`
- Create: `backend/internal/auth/handler.go`
- Create: `backend/internal/auth/handler_test.go`
- Create: `backend/internal/auth/routes.go`
- Create: `backend/platform/security/csrf.go`
- Create: `backend/platform/security/ratelimit.go`

**Interfaces:**
- Consumes: PasswordHasher and SessionService from Task 4
- Consumes: `MembershipResolver.ResolveActive(ctx, userID)` implemented in Task 6
- Produces: `POST /api/v1/auth/login`
- Produces: `POST /api/v1/auth/logout`
- Produces: `GET /api/v1/auth/session`
- Produces: `POST /api/v1/auth/change-password`
- Produces: Gin middleware `auth.RequireSession` and `security.RequireCSRF`

- [ ] **Step 1: Escribir prueba de error genérico y rate limit**

```go
func TestLoginDoesNotRevealAccountExistence(t *testing.T) {
    unknown := performLogin(t, router, "missing@example.com", "wrong")
    known := performLogin(t, router, "owner@example.com", "wrong")
    require.Equal(t, http.StatusUnauthorized, unknown.Code)
    require.JSONEq(t, unknown.Body.String(), known.Body.String())
}

func TestLoginRateLimit(t *testing.T) {
    for i := 0; i < 5; i++ {
        performLogin(t, router, "owner@example.com", "wrong")
    }
    res := performLogin(t, router, "owner@example.com", "wrong")
    require.Equal(t, http.StatusTooManyRequests, res.Code)
}
```

La respuesta debe ser `AUTH_INVALID_CREDENTIALS` para cuenta inexistente, inactiva o contraseña incorrecta.

- [ ] **Step 2: Implementar login y cookie**

```go
type LoginInput struct {
    Email string `json:"email" binding:"required,email,max=254"`
    Password string `json:"password" binding:"required,max=1048576"`
}

const SessionCookieName = "__Host-tallerflow_session"
```

Definir el puerto `MembershipResolver` dentro del servicio de login. El login solo crea sesión cuando devuelve exactamente una membresía activa. En desarrollo, permitir un nombre `tallerflow_session` solo cuando `Environment=development`; producción debe negarse a iniciar si `Secure` se desactiva.

- [ ] **Step 3: Escribir pruebas de sesión expirada y revocada**

```go
func TestRequireSessionClearsExpiredCookie(t *testing.T) {
    res := performWithExpiredSession(t, router)
    require.Equal(t, http.StatusUnauthorized, res.Code)
    require.Contains(t, res.Header().Get("Set-Cookie"), "Max-Age=0")
    require.Contains(t, res.Body.String(), "SESSION_INVALID")
}
```

Repetir para `revoked_at` no nulo.

- [ ] **Step 4: Escribir pruebas CSRF**

```go
func TestLogoutRejectsMissingCSRF(t *testing.T) {
    res := performAuthenticated(t, router, http.MethodPost, "/api/v1/auth/logout", nil, "")
    require.Equal(t, http.StatusForbidden, res.Code)
    require.Contains(t, res.Body.String(), "CSRF_INVALID")
}
```

Probar además origen ajeno y token correcto. Comparar tokens en tiempo constante.

- [ ] **Step 5: Implementar sesión y logout**

`GET /auth/session` devuelve fecha de expiración y CSRF raw únicamente para la sesión actual. `POST /logout` revoca en PostgreSQL y expira la cookie.

- [ ] **Step 6: Probar cambio obligatorio de contraseña**

```go
func TestChangePasswordRevokesOtherSessionsAndRotatesCurrent(t *testing.T) {
    res := performChangePassword(t, router, currentSession, "Temporary secure passphrase 9!", "New owner passphrase 10!")
    require.Equal(t, http.StatusOK, res.Code)
    require.NotEqual(t, currentSession.Token, sessionTokenFrom(res))
    requireOtherSessionsRevoked(t, userID)
    require.False(t, credentialMustChange(t, userID))
}
```

Rechazar contraseña actual incorrecta, nueva contraseña menor de 12 caracteres y reutilización de la contraseña vigente.

- [ ] **Step 7: Verificar y confirmar**

```powershell
cd backend
go test -race ./internal/auth ./platform/security ./platform/httpx
git add internal/auth platform/security platform/httpx
git commit -m "feat(auth): expose secure cookie session flow"
```

### Task 6: Workshop principal, memberships and authorization

**Owner:** Michael  
**Branch:** `feat/s1-michael-tenancy-team`

**Files:**
- Create: `backend/internal/workshops/domain.go`
- Create: `backend/internal/workshops/repository.go`
- Create: `backend/internal/workshops/service.go`
- Create: `backend/internal/workshops/service_test.go`
- Create: `backend/internal/workshops/handler.go`
- Create: `backend/internal/workshops/handler_test.go`
- Create: `backend/internal/workshops/routes.go`
- Create: `backend/platform/httpx/principal.go`

**Interfaces:**
- Consumes: authenticated user ID from Task 5
- Produces: `httpx.Principal{UserID, WorkshopID, Role}`
- Produces: `GET /api/v1/me`
- Produces: `GET /api/v1/workshops/current`
- Produces: `RequireRoles(roles ...Role) gin.HandlerFunc`

- [ ] **Step 1: Definir tipos compartidos**

```go
type Role string
const (
    RoleOwner Role = "OWNER"
    RoleAdmin Role = "ADMIN"
    RoleOperator Role = "OPERATOR"
)

type Principal struct {
    UserID uuid.UUID
    WorkshopID uuid.UUID
    Role Role
}
```

- [ ] **Step 2: Probar membresía activa**

```go
func TestResolvePrincipalRejectsInactiveMembership(t *testing.T) {
    repo := fakeMembershipRepo{membership: Membership{Status: StatusInactive}}
    _, err := NewService(repo).ResolvePrincipal(ctx, userID)
    require.ErrorIs(t, err, ErrNoActiveMembership)
}
```

El MVP exige exactamente una membresía activa por usuario. Cero o más de una producen error controlado y evento de auditoría.

- [ ] **Step 3: Implementar repositorio tenant-aware**

```go
type MembershipRepository interface {
    FindActiveByUser(context.Context, uuid.UUID) ([]Membership, error)
}

type Service struct{ memberships MembershipRepository }
func (s Service) ResolvePrincipal(ctx context.Context, userID uuid.UUID) (Principal, error)
```

El adaptador GORM invoca `SELECT * FROM resolve_active_memberships(?)`; no desactiva RLS ni consulta `workshop_members` directamente sin contexto.

- [ ] **Step 4: Probar que el principal solo resuelve su taller**

```go
func TestCurrentWorkshopUsesPrincipalTenant(t *testing.T) {
    res := performAsWorkshop(t, router, workshopA, http.MethodGet, "/api/v1/workshops/current")
    require.Equal(t, http.StatusOK, res.Code)
    require.Contains(t, res.Body.String(), workshopA.ID.String())
    require.NotContains(t, res.Body.String(), workshopB.ID.String())
}
```

- [ ] **Step 5: Implementar endpoints y autorización**

`/me` devuelve `id`, `name`, `email`, `workshop{id,name,timezone}`, `role` y `must_change_password`. No devuelve CSRF, hashes ni IDs de sesiones.

- [ ] **Step 6: Verificar y confirmar**

```powershell
cd backend
go test -race ./internal/workshops ./platform/httpx
git add internal/workshops platform/httpx
git commit -m "feat(workshops): resolve tenant principal and roles"
```

### Task 7: OWNER bootstrap and audit

**Owner:** José  
**Branch:** `feat/s1-jose-auth-api`

**Files:**
- Create: `backend/cmd/bootstrap/main.go`
- Create: `backend/internal/auth/bootstrap.go`
- Create: `backend/internal/auth/bootstrap_test.go`
- Create: `backend/internal/audit/event.go`
- Create: `backend/internal/audit/repository.go`

**Interfaces:**
- Consumes: PasswordHasher, PostgreSQL and workshop models
- Produces: CLI `go run ./cmd/bootstrap --email ... --name ... --workshop ...`

- [ ] **Step 1: Escribir prueba transaccional**

```go
func TestBootstrapCreatesOwnerAtomically(t *testing.T) {
    result, err := service.CreateOwner(ctx, BootstrapInput{
        Email: "owner@tallerflow.pe",
        Name: "Owner Demo",
        WorkshopName: "Taller Demo",
        Password: "Temporary secure passphrase 9!",
    })
    require.NoError(t, err)
    require.Equal(t, workshops.RoleOwner, result.Role)
    require.True(t, result.MustChangePassword)
    requireAuditEvent(t, "OWNER_BOOTSTRAPPED")
}
```

Probar duplicado de email y rollback cuando falla la membresía.

- [ ] **Step 2: Implementar caso de uso**

```go
type BootstrapInput struct {
    Email string
    Name string
    WorkshopName string
    Password string
}

func (s BootstrapService) CreateOwner(ctx context.Context, in BootstrapInput) (BootstrapResult, error)
```

Crear usuario, credencial, taller, membresía OWNER y auditoría en una transacción administrativa.

- [ ] **Step 3: Implementar CLI sin filtrar contraseña**

La contraseña entra por `TF_BOOTSTRAP_PASSWORD`, no por argumento ni log. Los argumentos solo contienen email, nombre y taller.

```powershell
$env:TF_BOOTSTRAP_PASSWORD = "temporary-value"
go run ./cmd/bootstrap --email owner@tallerflow.pe --name "Owner Demo" --workshop "Taller Demo"
```

- [ ] **Step 4: Probar idempotencia segura**

Una segunda ejecución devuelve `BOOTSTRAP_ALREADY_EXISTS`, no modifica contraseña ni taller y termina con código distinto de cero.

- [ ] **Step 5: Verificar y confirmar**

```powershell
cd backend
go test -race ./internal/auth ./internal/audit ./cmd/bootstrap
git add cmd/bootstrap internal/auth internal/audit
git commit -m "feat(auth): provision initial workshop owner"
```

### Task 8: Vue TypeScript shell and design system

**Owner:** Lucero  
**Branch:** `feat/s1-lucero-frontend-shell`

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/src/app/main.ts`
- Create: `frontend/src/app/router.ts`
- Create: `frontend/src/app/query.ts`
- Create: `frontend/vite.config.ts`
- Create: `frontend/src/styles/tokens.css`
- Create: `frontend/src/styles/main.css`
- Create: `frontend/src/components/ui/AppButton.vue`
- Create: `frontend/src/components/ui/AppField.vue`
- Create: `frontend/src/components/ui/AppAlert.vue`
- Create: `frontend/src/layouts/AuthLayout.vue`
- Create: `frontend/src/components/ui/AppButton.spec.ts`

**Interfaces:**
- Produces: aliases `@/*`
- Produces: named routes `login` and `dashboard`
- Produces: UI primitives used by Task 9

- [ ] **Step 1: Crear proyecto TypeScript**

```powershell
npm create vue@latest frontend
# Seleccionar TypeScript, Router, Pinia, Vitest, ESLint y Prettier.
cd frontend
npm install
npm install @tanstack/vue-query tailwindcss @tailwindcss/vite reka-ui vee-validate zod @vee-validate/zod
npm install -D msw @playwright/test openapi-typescript
```

Eliminar componentes de demostración y conservar el lockfile.

Agregar `generate:api` para producir `src/types/api.generated.ts` desde `../docs/contracts/openapi.yaml`. Configurar el proxy de Vite para `/api` y `/health` hacia `http://localhost:8080`.

- [ ] **Step 2: Escribir prueba fallida del botón**

```ts
it('blocks a second submit while loading', async () => {
  const wrapper = mount(AppButton, {
    props: { loading: true },
    slots: { default: 'Ingresar' },
  })
  expect(wrapper.get('button').attributes('disabled')).toBeDefined()
  expect(wrapper.text()).toContain('Ingresando')
})
```

- [ ] **Step 3: Implementar tokens y componentes**

```css
:root {
  --tf-color-brand-600: #087f67;
  --tf-color-brand-500: #0fae87;
  --tf-color-ink: #0b1736;
  --tf-color-muted: #60708f;
  --tf-color-danger: #c9364f;
  --tf-radius-card: 1rem;
  --tf-shadow-card: 0 12px 32px rgb(11 23 54 / 0.08);
}
```

`AppButton` admite `variant: primary|secondary|danger`, `loading` y `disabled`. `AppField` conecta label, descripción y error mediante IDs.

- [ ] **Step 4: Crear shell y rutas lazy**

```ts
const routes: RouteRecordRaw[] = [
  { path: '/login', name: 'login', component: () => import('@/modules/auth/LoginView.vue') },
  { path: '/', name: 'dashboard', component: () => import('@/modules/dashboard/DashboardPlaceholder.vue'), meta: { requiresAuth: true } },
]
```

- [ ] **Step 5: Verificar y confirmar**

```powershell
cd frontend
npm run type-check
npm run lint
npm run test:unit -- --run
npm run build
git add frontend
git commit -m "feat(frontend): add Vue TypeScript design foundation"
```

### Task 9: Frontend session and login

**Owner:** Lucero  
**Branch:** `feat/s1-lucero-frontend-shell`

**Files:**
- Create: `frontend/src/services/http.ts`
- Create: `frontend/src/modules/session/types.ts`
- Create: `frontend/src/modules/session/api.ts`
- Create: `frontend/src/modules/session/store.ts`
- Create: `frontend/src/modules/session/store.spec.ts`
- Create: `frontend/src/modules/auth/api.ts`
- Create: `frontend/src/modules/auth/LoginView.vue`
- Create: `frontend/src/modules/auth/LoginView.spec.ts`
- Create: `frontend/src/modules/auth/ChangePasswordView.vue`
- Create: `frontend/src/modules/auth/ChangePasswordView.spec.ts`
- Create: `frontend/src/test/server.ts`
- Create: `frontend/src/test/fixtures.ts`
- Modify: `frontend/src/app/router.ts`

**Interfaces:**
- Consumes: `GET /auth/session`, `POST /auth/login`, `POST /auth/logout`
- Produces: `useSessionStore().restore/login/logout`
- Produces: `useSessionStore().changePassword`
- Produces: route guard based on `requiresAuth`

- [ ] **Step 1: Implementar cliente HTTP tipado**

```ts
export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...init,
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', ...init.headers },
  })
  if (!response.ok) throw await ApiError.fromResponse(response)
  return (await response.json()).data as T
}
```

Las mutaciones leen CSRF desde el store en memoria y lo envían como `X-CSRF-Token`.

- [ ] **Step 2: Escribir prueba de restauración**

```ts
it('restores a server session without browser storage', async () => {
  server.use(
    http.get('/api/v1/auth/session', () => HttpResponse.json({ data: sessionFixture })),
    http.get('/api/v1/me', () => HttpResponse.json({ data: meFixture })),
  )
  const store = useSessionStore()
  await store.restore()
  expect(store.user?.email).toBe('owner@tallerflow.pe')
  expect(localStorage.length).toBe(0)
})
```

- [ ] **Step 3: Implementar store**

```ts
export const useSessionStore = defineStore('session', () => {
  const status = ref<'unknown' | 'authenticated' | 'guest'>('unknown')
  const user = ref<SessionUser | null>(null)
  const csrfToken = ref<string | null>(null)
  async function restore(): Promise<void>
  async function login(input: LoginInput): Promise<void>
  async function logout(): Promise<void>
  return { status, user, csrfToken, restore, login, logout }
})
```

- [ ] **Step 4: Escribir prueba de LoginView**

```ts
it('shows the generic credential error and enables retry', async () => {
  server.use(http.post('/api/v1/auth/login', () =>
    HttpResponse.json({ error: { code: 'AUTH_INVALID_CREDENTIALS', message: 'Correo o contraseña incorrectos.' } }, { status: 401 }),
  ))
  const wrapper = mountLogin()
  await wrapper.get('[name=email]').setValue('owner@tallerflow.pe')
  await wrapper.get('[name=password]').setValue('incorrect')
  await wrapper.get('form').trigger('submit')
  await flushPromises()
  expect(wrapper.text()).toContain('Correo o contraseña incorrectos')
  expect(wrapper.get('button[type=submit]').attributes('disabled')).toBeUndefined()
})
```

- [ ] **Step 5: Implementar login y guard**

El formulario usa Zod, no muestra si una cuenta existe, bloquea doble submit y permite mostrar/ocultar contraseña. Si `must_change_password` es verdadero, redirige a `/change-password`; de lo contrario continúa a la ruta solicitada.

- [ ] **Step 6: Implementar cambio obligatorio**

`ChangePasswordView` solicita contraseña actual, nueva contraseña y confirmación. Tras éxito reemplaza la sesión rotada en memoria y navega al dashboard. No permite abandonar hacia rutas privadas mientras `must_change_password` siga activo.

- [ ] **Step 7: Verificar responsive, teclado y commit**

```powershell
cd frontend
npm run type-check
npm run lint
npm run test:unit -- --run
npm run build
git add src
git commit -m "feat(frontend-auth): add secure login and session restore"
```

### Task 10: Integrated Compose, Caddy and CI

**Owner:** Stefano  
**Branch:** `feat/s1-stefano-data-platform`

**Files:**
- Create: `backend/Dockerfile`
- Create: `frontend/Dockerfile`
- Create: `infra/caddy/Caddyfile`
- Modify: `compose.yaml`
- Create: `compose.production.yaml`
- Create: `.github/workflows/ci.yml`
- Modify: `README.md`

**Interfaces:**
- Consumes: backend and frontend build commands
- Produces: `docker compose up --build`
- Produces: CI jobs `database`, `backend`, `frontend`

- [ ] **Step 1: Crear imágenes multi-stage**

```dockerfile
# backend/Dockerfile
FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/tallerflow-api ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -o /out/tallerflow-bootstrap ./cmd/bootstrap
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/tallerflow-api /tallerflow-api
COPY --from=build /out/tallerflow-bootstrap /tallerflow-bootstrap
ENTRYPOINT ["/tallerflow-api"]
```

El Dockerfile frontend usa `node:24.21.0-alpine`, ejecuta `npm ci && npm run build` y entrega `dist` a la imagen de Caddy.

- [ ] **Step 2: Crear Caddy same-origin**

```caddyfile
:80 {
  encode zstd gzip
  handle /health/* {
    reverse_proxy backend:8080
  }
  handle /api/* {
    reverse_proxy backend:8080
  }
  handle {
    root * /srv
    try_files {path} /index.html
    file_server
  }
}
```

- [ ] **Step 3: Completar Compose**

Orden: PostgreSQL saludable, Flyway completado, backend ready, frontend/Caddy. No publicar PostgreSQL fuera de localhost en desarrollo ni exponerlo en producción.

- [ ] **Step 4: Crear CI**

```yaml
jobs:
  backend:
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v6
        with: { go-version: '1.27.1' }
      - run: cd backend && go test -race ./... && go vet ./...
  frontend:
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v5
        with: { node-version: '24.21.0', cache: npm, cache-dependency-path: frontend/package-lock.json }
      - run: cd frontend && npm ci
      - run: cd frontend && npm run type-check && npm run lint && npm run test:unit -- --run && npm run build
```

El job database arranca PostgreSQL 18.6 y ejecuta `flyway validate`, `migrate` y las verificaciones SQL.

- [ ] **Step 5: Probar desde limpio y confirmar**

```powershell
docker compose config
docker compose up --build -d
Invoke-WebRequest http://localhost/health/live
Invoke-WebRequest http://localhost/health/ready
git add backend/Dockerfile frontend/Dockerfile infra compose.yaml compose.production.yaml .github README.md
git commit -m "ci: integrate Docker Caddy and validation pipeline"
```

### Task 11: End-to-end security flow

**Owners:** José integra; Lucero, Michael y Stefano revisan sus capas  
**Branch de integración:** `feat/s1-integration-foundations`, creada desde `main` después de fusionar las cuatro ramas personales

**Files:**
- Create: `frontend/e2e/auth.spec.ts`
- Create: `backend/internal/auth/auth_integration_test.go`
- Create: `docs/runbooks/local-development.md`
- Create: `docs/runbooks/bootstrap-owner.md`

**Interfaces:**
- Consumes: Tasks 1–10
- Produces: recorrido integrado y repetible

- [ ] **Step 1: Escribir E2E de login**

```ts
test('owner logs in, sees workshop and logs out', async ({ page }) => {
  await page.goto('/login')
  await page.getByLabel('Correo').fill('owner@tallerflow.pe')
  await page.getByLabel('Contraseña').fill(process.env.E2E_OWNER_PASSWORD!)
  await page.getByRole('button', { name: 'Ingresar' }).click()
  await expect(page).toHaveURL(/\/change-password$/)
  await page.getByLabel('Contraseña actual').fill(process.env.E2E_OWNER_PASSWORD!)
  await page.getByLabel('Nueva contraseña').fill(process.env.E2E_NEW_OWNER_PASSWORD!)
  await page.getByLabel('Confirmar contraseña').fill(process.env.E2E_NEW_OWNER_PASSWORD!)
  await page.getByRole('button', { name: 'Cambiar contraseña' }).click()
  await expect(page.getByText('Taller Demo')).toBeVisible()
  await page.getByRole('button', { name: 'Cerrar sesión' }).click()
  await expect(page).toHaveURL(/\/login$/)
})
```

- [ ] **Step 2: Añadir casos negativos**

Playwright debe comprobar credenciales inválidas, doble envío, sesión expirada y responsive a 360 px. La integración Go debe comprobar cookie segura, CSRF ausente, origen ajeno, rate limit y RLS cruzado.

- [ ] **Step 3: Ejecutar recorrido limpio**

```powershell
docker compose down -v
docker compose up --build -d
$env:TF_BOOTSTRAP_PASSWORD = $env:E2E_OWNER_PASSWORD
docker compose exec backend /tallerflow-bootstrap --email owner@tallerflow.pe --name "Owner Demo" --workshop "Taller Demo"
cd frontend
npx playwright test e2e/auth.spec.ts
```

- [ ] **Step 4: Ejecutar la puerta completa**

```powershell
cd backend
go test -race ./...
go vet ./...
cd ..\frontend
npm run type-check
npm run lint
npm run test:unit -- --run
npm run build
npx playwright test
```

- [ ] **Step 5: Documentar y confirmar**

Los runbooks deben contener prerrequisitos, variables sin secretos reales, arranque, bootstrap, pruebas, solución para puertos ocupados y limpieza de volúmenes.

```powershell
git add backend frontend docs/runbooks
git commit -m "test: verify secure owner session end to end"
```

## Merge order

1. Stefano: PostgreSQL, Flyway y esquema.
2. José: backend base, crypto y sesiones.
3. Michael: tenant runner, principal y roles.
4. Lucero: Vue shell y login.
5. Stefano: Compose, Caddy y CI integrados.
6. José: rama de integración y E2E.

## Phase 1 acceptance

- `docker compose up --build` funciona desde limpio.
- Flyway valida y migra una base vacía dos veces sin cambios.
- Bootstrap crea un único OWNER y exige cambio de contraseña.
- El primer login obliga a cambiar la contraseña y rota la sesión.
- Login no revela si una cuenta existe.
- La cookie y CSRF cumplen la especificación.
- Una sesión expirada o revocada devuelve `401`.
- RLS bloquea un taller ajeno y ausencia de contexto.
- Vue no conserva tokens ni credenciales en almacenamiento web.
- Login funciona con teclado y a 360 px.
- Backend, frontend, database y E2E pasan en CI.
