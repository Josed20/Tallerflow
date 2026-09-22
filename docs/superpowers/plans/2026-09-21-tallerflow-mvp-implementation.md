# TallerFlow MVP Implementation Plan (OBSOLETO)

> **No ejecutar este plan.** Corresponde a la arquitectura anterior con Supabase y JavaScript. Se regenerará después de aprobar la nueva especificación.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Entregar un MVP desplegable de TallerFlow que permita crear una orden textil, registrar su avance por etapas y mostrar al cliente un tracking privado.

**Architecture:** Un monolito modular en Go expone una API REST multi-tenant; Vue 3 consume exclusivamente esa API para las experiencias de dueño, administrador, operario y cliente externo. PostgreSQL conserva el estado y el historial append-only, Flyway es la única autoridad del esquema, y Supabase aporta autenticación y almacenamiento de evidencias.

**Tech Stack:** Go, Gin, GORM, PostgreSQL, Flyway, Vue 3, Vite, JavaScript, Pinia, Vue Router, Axios, Tailwind CSS, Vitest, Vue Test Utils, Playwright, Docker Compose, Supabase Auth y Supabase Storage.

**Spec:** `docs/superpowers/specs/2026-09-21-tallerflow-mvp-design.md`

## Global Constraints

- No usar `GORM AutoMigrate`; todo cambio de esquema se hace mediante una nueva migración Flyway forward-only.
- Todos los identificadores persistidos son UUID; el código visible de orden es secuencial por taller.
- Toda consulta interna de negocio incluye `workshop_id`; un recurso de otro taller responde `404`.
- El historial de producción es append-only; una corrección agrega un evento y nunca reescribe el original.
- Los timestamps se almacenan en UTC, las fechas comprometidas usan `DATE` y el semáforo se calcula en `America/Lima`.
- El tracking público usa un token aleatorio de 256 bits como mínimo, no requiere JWT y devuelve un DTO público separado.
- Las evidencias nacen internas, admiten JPEG, PNG o WebP hasta 10 MB y solo `OWNER` o `ADMIN` pueden publicarlas.
- No incorporar Redis, Kafka, WebSockets, microservicios, CQRS, event sourcing, trabajo offline ni aplicaciones móviles nativas.
- Mantener dependencias externas detrás de interfaces para probar sin llamar a Supabase.
- Usar TDD por tarea, ejecutar la comprobación indicada y crear un commit pequeño al finalizar cada tarea.

## Review Focus

- UUID válido perteneciente a otro taller: la API debe responder `404`, cubierto en las pruebas de autorización de las tareas 4, 5 y 6.
- Dos avances simultáneos con el mismo `client_request_id`: debe persistirse un solo evento y devolverse el resultado idempotente, cubierto en la tarea 9.
- Archivo con extensión permitida pero MIME o contenido falso, o tamaño mayor a 10 MB: debe rechazarse sin borrar el avance, cubierto en la tarea 11.
- Fecha de entrega evaluada alrededor de medianoche UTC: debe usar el día de Lima, cubierto en la tarea 13.
- Dos solicitudes que intentan avanzar, pausar o cancelar la misma etapa: una debe confirmar y la otra devolver `409`, cubierto en las tareas 8 y 10.

---

## Mapa de archivos y fases

```text
Fase 1 — Base ejecutable
  backend/cmd/server                 composición y arranque HTTP
  backend/internal/platform         config, DB, HTTP, logs y pruebas auxiliares
  database/migrations               esquema Flyway
  frontend/src                      shell Vue, router, sesión y cliente HTTP
  compose.yaml                      entorno local reproducible

Fase 2 — Primer corte vertical
  backend/internal/auth             JWT, principal y tenant
  backend/internal/clients          clientes del taller
  backend/internal/orders           crear/listar órdenes y etapas iniciales
  frontend/src/modules/clients      clientes
  frontend/src/modules/orders       listado y creación de órdenes

Fase 3 — Trazabilidad productiva
  backend/internal/production       detalle, avances, etapas y estados
  backend/internal/attachments      evidencias y Supabase Storage
  frontend/src/modules/production   detalle del dueño y captura del operario

Fase 4 — Visibilidad y administración
  backend/internal/dashboard        resumen operativo
  backend/internal/tracking         proyección pública
  backend/internal/team             invitaciones y membresías
  frontend/src/modules/dashboard    tablero
  frontend/src/modules/tracking     consulta externa
  frontend/src/modules/team         equipo

Fase 5 — Cierre del MVP
  backend/internal/platform         endurecimiento operativo
  frontend/e2e                      flujos Playwright
  .github/workflows/ci.yml          validación completa
  README.md                          operación local y despliegue
```

## Fase 1 — Base ejecutable

### Task 1: Backend mínimo, configuración y health checks

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/server/main.go`
- Create: `backend/internal/platform/config/config.go`
- Create: `backend/internal/platform/config/config_test.go`
- Create: `backend/internal/platform/httpx/router.go`
- Create: `backend/internal/platform/httpx/health.go`
- Create: `backend/internal/platform/httpx/health_test.go`
- Create: `backend/internal/platform/database/postgres.go`
- Create: `backend/.env.example`
- Create: `backend/Dockerfile`

**Interfaces:**
- Consumes: variables `APP_ENV`, `HTTP_ADDR`, `DATABASE_URL`, `SUPABASE_URL`, `SUPABASE_JWT_ISSUER`, `SUPABASE_JWT_AUDIENCE`, `SUPABASE_SERVICE_ROLE_KEY`, `SUPABASE_STORAGE_BUCKET`, `CORS_ORIGINS`.
- Produces: `config.Load() (config.Config, error)`, `database.Open(string) (*gorm.DB, error)`, `httpx.NewRouter(httpx.Dependencies) *gin.Engine` y health checks `/health/live`, `/health/ready`.

- [ ] **Step 1: Inicializar el módulo y escribir pruebas fallidas de configuración**

Run first:

```powershell
cd backend
go mod init github.com/Josed20/Tallerflow/backend
go get github.com/gin-gonic/gin gorm.io/gorm gorm.io/driver/postgres github.com/google/uuid github.com/stretchr/testify
```

```go
func TestLoadRejectsMissingDatabaseURL(t *testing.T) {
    t.Setenv("DATABASE_URL", "")
    _, err := Load()
    require.ErrorContains(t, err, "DATABASE_URL")
}

func TestLoadDefaults(t *testing.T) {
    setRequiredEnv(t)
    cfg, err := Load()
    require.NoError(t, err)
    assert.Equal(t, ":8080", cfg.HTTPAddr)
    assert.Equal(t, "America/Lima", cfg.BusinessTimezone)
}
```

- [ ] **Step 2: Ejecutar las pruebas y confirmar el fallo inicial**

Run: `cd backend; go test ./internal/platform/config ./internal/platform/httpx`

Expected: FAIL porque `Load` y `NewRouter` todavía no existen.

- [ ] **Step 3: Implementar configuración tipada, conexión y router mínimo**

```go
type Config struct {
    AppEnv, HTTPAddr, DatabaseURL, SupabaseURL string
    SupabaseJWTIssuer, SupabaseJWTAudience string
    SupabaseServiceRoleKey, SupabaseStorageBucket string
    CORSOrigins []string
    BusinessTimezone string
}

type Dependencies struct {
    DB *gorm.DB
}
```

`/health/live` devuelve `200 {"data":{"status":"ok"}}`. `/health/ready` ejecuta `SELECT 1`; devuelve `200` si PostgreSQL responde y `503` en caso contrario. `main.go` carga configuración, abre GORM con el driver PostgreSQL, construye el router y usa `http.Server` con timeouts explícitos.

- [ ] **Step 4: Verificar backend y análisis estático**

Run: `cd backend; go test ./...; go vet ./...`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend
git commit -m "feat: bootstrap Go API and health checks"
```

### Task 2: PostgreSQL, Flyway y esquema inicial completo

**Files:**
- Create: `compose.yaml`
- Create: `database/migrations/V1__create_extensions.sql`
- Create: `database/migrations/V2__create_identity_and_workshops.sql`
- Create: `database/migrations/V3__create_clients.sql`
- Create: `database/migrations/V4__create_orders_and_stages.sql`
- Create: `database/migrations/V5__create_updates_and_attachments.sql`
- Create: `database/migrations/V6__add_constraints_and_indexes.sql`
- Create: `backend/internal/platform/database/schema_integration_test.go`
- Create: `.env.example`

**Interfaces:**
- Consumes: `database.Open` de la tarea 1 y PostgreSQL expuesto solo en desarrollo.
- Produces: las diez tablas del diseño, enums mediante `CHECK`, claves foráneas, índices tenant-aware y el servicio Compose `flyway` que termina antes de iniciar `backend`.

- [ ] **Step 1: Escribir la prueba de contrato del esquema antes de las migraciones**

```go
func TestMigratedSchemaHasCoreTablesAndIndexes(t *testing.T) {
    db := openIntegrationDB(t)
    for _, table := range []string{"users", "workshops", "workshop_users", "user_invitations", "clients", "workshop_counters", "orders", "order_stages", "order_updates", "attachments"} {
        assert.True(t, db.Migrator().HasTable(table), table)
    }
    assert.True(t, hasIndex(t, db, "order_updates", "ux_order_updates_request"))
    assert.True(t, hasIndex(t, db, "order_stages", "ux_order_stages_current"))
}
```

- [ ] **Step 2: Levantar PostgreSQL sin migraciones y confirmar el fallo**

Run: `docker compose up -d postgres; cd backend; $env:TEST_DATABASE_URL='postgres://tallerflow:tallerflow@localhost:5432/tallerflow?sslmode=disable'; go test ./internal/platform/database -run TestMigratedSchema`

Expected: FAIL porque las tablas no existen.

- [ ] **Step 3: Crear las seis migraciones**

Las migraciones deben materializar exactamente las columnas de la especificación. `V6` agrega, como mínimo:

```sql
CREATE UNIQUE INDEX ux_order_updates_request
ON order_updates (order_id, client_request_id)
WHERE client_request_id IS NOT NULL;

CREATE UNIQUE INDEX ux_order_stages_current
ON order_stages (order_id)
WHERE status IN ('IN_PROGRESS', 'BLOCKED');

CREATE INDEX ix_orders_workshop_status_delivery
ON orders (workshop_id, status, delivery_date);

CREATE INDEX ix_updates_workshop_order_created
ON order_updates (workshop_id, order_id, created_at DESC);
```

Agregar restricciones para cantidad positiva, fechas válidas, roles, estados de membresía, orden, etapa, invitación y tipos de evento. La FK diferida `orders.current_stage_id -> order_stages.id` se agrega en `V6` para resolver el ciclo de creación.

- [ ] **Step 4: Ejecutar Flyway y probar una base vacía**

Run: `docker compose run --rm flyway migrate; docker compose run --rm flyway validate; cd backend; go test ./internal/platform/database -run TestMigratedSchema`

Expected: Flyway aplica V1–V6, `validate` termina correctamente y la prueba pasa.

- [ ] **Step 5: Probar repetibilidad operativa**

Run: `docker compose run --rm flyway migrate`

Expected: `Schema ... is up to date. No migration necessary.`

- [ ] **Step 6: Commit**

```bash
git add compose.yaml .env.example database backend/internal/platform/database/schema_integration_test.go
git commit -m "feat: define TallerFlow schema with Flyway"
```

### Task 3: Shell Vue, sistema visual y cliente HTTP

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.js`
- Create: `frontend/vitest.config.js`
- Create: `frontend/tailwind.config.js`
- Create: `frontend/postcss.config.js`
- Create: `frontend/index.html`
- Create: `frontend/src/main.js`
- Create: `frontend/src/App.vue`
- Create: `frontend/src/assets/main.css`
- Create: `frontend/src/router/index.js`
- Create: `frontend/src/layouts/AppLayout.vue`
- Create: `frontend/src/components/base/BaseButton.vue`
- Create: `frontend/src/components/base/BaseField.vue`
- Create: `frontend/src/components/feedback/ViewState.vue`
- Create: `frontend/src/services/http.js`
- Create: `frontend/src/services/http.test.js`
- Create: `frontend/src/test/setup.js`
- Create: `frontend/.env.example`
- Create: `frontend/Dockerfile`
- Modify: `compose.yaml`

**Interfaces:**
- Consumes: API bajo `VITE_API_BASE_URL`.
- Produces: `http` Axios configurado, `setAccessTokenProvider(provider)`, `AppLayout`, componentes base y rutas con `meta.requiresAuth`/`meta.roles`.

- [ ] **Step 1: Escribir pruebas fallidas del cliente HTTP**

```js
it('envía el bearer token y conserva request_id del error', async () => {
  setAccessTokenProvider(async () => 'jwt-test')
  mock.onGet('/me').reply(500, { error: { code: 'X', request_id: 'req-1' } })
  await expect(http.get('/me')).rejects.toMatchObject({
    code: 'X', requestId: 'req-1'
  })
  expect(mock.history.get[0].headers.Authorization).toBe('Bearer jwt-test')
})
```

- [ ] **Step 2: Instalar dependencias y confirmar el fallo**

Run: `cd frontend; npm install; npm run test -- --run`

Expected: FAIL porque `src/services/http.js` no existe.

- [ ] **Step 3: Implementar shell y normalización de errores**

```js
export class ApiError extends Error {
  constructor(payload, status) {
    super(payload?.message ?? 'No pudimos completar la solicitud.')
    this.code = payload?.code ?? 'UNEXPECTED_ERROR'
    this.details = payload?.details ?? {}
    this.requestId = payload?.request_id
    this.status = status
  }
}
```

Configurar Axios para anteponer `/api/v1`, obtener el token mediante el proveedor inyectado, adjuntarlo y convertir el contrato `{error}` en `ApiError`. Construir los tokens visuales a partir de Stitch: azul marino para estructura, verde turquesa para acción/avance, fondos claros y controles táctiles de al menos 44 px.

- [ ] **Step 4: Integrar frontend en Compose y verificar**

Run: `cd frontend; npm run test -- --run; npm run build; cd ..; docker compose config`

Expected: pruebas, build y validación de Compose exitosos.

- [ ] **Step 5: Commit**

```bash
git add frontend compose.yaml
git commit -m "feat: bootstrap Vue application shell"
```

## Fase 2 — Primer corte vertical autenticado

### Task 4: Autenticación Supabase, principal local y aislamiento tenant

**Files:**
- Create: `backend/internal/auth/model.go`
- Create: `backend/internal/auth/repository.go`
- Create: `backend/internal/auth/service.go`
- Create: `backend/internal/auth/jwt.go`
- Create: `backend/internal/auth/middleware.go`
- Create: `backend/internal/auth/handler.go`
- Create: `backend/internal/auth/middleware_test.go`
- Create: `backend/internal/auth/service_integration_test.go`
- Create: `backend/cmd/bootstrap/main.go`
- Modify: `backend/internal/platform/httpx/router.go`
- Create: `frontend/src/modules/auth/services/supabase.js`
- Create: `frontend/src/modules/auth/stores/session.js`
- Create: `frontend/src/modules/auth/views/LoginView.vue`
- Create: `frontend/src/modules/auth/views/LoginView.test.js`
- Modify: `frontend/src/router/index.js`

**Interfaces:**
- Consumes: Supabase JWT con firma, `iss`, `aud` y `exp` válidos; tablas `users`, `workshops`, `workshop_users`, `user_invitations`.
- Produces: `Authenticator.Verify(ctx, token) (Identity, error)`, `Service.ResolvePrincipal(ctx, authUserID) (Principal, error)`, `auth.Require(service) gin.HandlerFunc`, `auth.RequireRoles(roles ...Role) gin.HandlerFunc`, `GET /api/v1/me` y `GET /api/v1/workshops/current`.

- [ ] **Step 1: Escribir pruebas de JWT, roles y tenant**

```yaml
- test: TestRequireRejectsExpiredToken
  given: "Authorization: Bearer expired; verifier returns ErrExpired"
  when: "GET /api/v1/me"
  expect: "401 and error.code=AUTH_TOKEN_EXPIRED"
- test: TestRequireRolesRejectsOperator
  given: "authenticated Principal with role OPERATOR"
  when: "POST /api/v1/clients"
  expect: "403 and error.code=ROLE_FORBIDDEN"
- test: TestResolvePrincipalRejectsInactiveMembership
  given: "valid identity with membership status INACTIVE"
  when: "ResolvePrincipal is called"
  expect: "ErrForbidden"
- test: TestTenantLookupReturnsNotFoundForForeignWorkshop
  given: "resource belongs to workshop B and principal belongs to workshop A"
  when: "tenant repository lookup uses resource UUID"
  expect: "ErrNotFound, never ErrForbidden"
```

- [ ] **Step 2: Ejecutar pruebas y confirmar el fallo**

Run: `cd backend; go test ./internal/auth`

Expected: FAIL por interfaces y middleware inexistentes.

- [ ] **Step 3: Implementar autenticación y contexto**

Run: `cd backend; go get github.com/lestrrat-go/jwx/v3`

```go
type Role string
const (
    RoleOwner Role = "OWNER"
    RoleAdmin Role = "ADMIN"
    RoleOperator Role = "OPERATOR"
)

type Principal struct {
    UserID, AuthUserID, WorkshopID uuid.UUID
    Role Role
}
```

El autenticador carga y refresca el JWKS publicado por Supabase, exige el algoritmo declarado por la clave, y valida firma, `iss`, `aud` y `exp`. El middleware extrae un único bearer token, lo verifica, resuelve membresía activa y almacena `Principal` en el contexto Gin. El bootstrap recibe email, nombre y taller por flags, busca el usuario Supabase y crea de forma transaccional `users`, `workshops`, `workshop_users(OWNER)` y `workshop_counters`.

- [ ] **Step 4: Implementar login Vue y guardas**

La store expone `signIn(email, password)`, `restore()`, `loadMe()` y `signOut()`, y registra en `http` un proveedor que obtiene `session.access_token` desde el cliente oficial de Supabase. La guarda redirige a `/login` sin sesión y a `/403` si el rol no está permitido. El test de `LoginView` simula error de credenciales y navegación exitosa.

- [ ] **Step 5: Ejecutar pruebas de ambas capas**

Run: `cd backend; go test ./internal/auth; cd ../frontend; npm run test -- --run src/modules/auth`

Expected: PASS, incluyendo el `404` tenant-aware.

- [ ] **Step 6: Commit**

```bash
git add backend frontend
git commit -m "feat: add Supabase authentication and tenant context"
```

### Task 5: Clientes del taller

**Files:**
- Create: `backend/internal/clients/model.go`
- Create: `backend/internal/clients/dto.go`
- Create: `backend/internal/clients/repository.go`
- Create: `backend/internal/clients/service.go`
- Create: `backend/internal/clients/handler.go`
- Create: `backend/internal/clients/service_test.go`
- Create: `backend/internal/clients/handler_test.go`
- Create: `backend/internal/clients/repository_integration_test.go`
- Modify: `backend/internal/platform/httpx/router.go`
- Create: `frontend/src/modules/clients/services/clients.js`
- Create: `frontend/src/modules/clients/views/ClientsView.vue`
- Create: `frontend/src/modules/clients/components/ClientForm.vue`
- Create: `frontend/src/modules/clients/components/ClientForm.test.js`

**Interfaces:**
- Consumes: `auth.Principal` y tabla `clients`.
- Produces: `clients.Service.List/Create/Get/Update`, rutas `GET|POST /clients`, `GET|PATCH /clients/:id` y selector reutilizable `ClientForm`.

- [ ] **Step 1: Escribir pruebas de validación y aislamiento**

```yaml
- test: TestCreateRejectsBlankName
  input: {name: "   "}
  expect: "ErrValidation with details.name=required"
- test: TestGetForeignClientReturnsNotFound
  given: "client.workshop_id differs from principal.workshop_id"
  expect: "404 CLIENT_NOT_FOUND"
- test: TestOperatorCannotCreateClient
  given: "role OPERATOR"
  expect: "403 ROLE_FORBIDDEN"
- test: TestUpdateDeactivatesWithoutDeleting
  input: {is_active: false}
  expect: "response.is_active=false and SELECT COUNT(*) for the UUID remains 1"
```

- [ ] **Step 2: Ejecutar el paquete y observar el fallo**

Run: `cd backend; go test ./internal/clients`

Expected: FAIL porque el módulo no existe.

- [ ] **Step 3: Implementar repositorio, servicio, DTO y handlers**

```go
type CreateInput struct { Name, CompanyName, Phone, Email string }
type UpdateInput struct { Name, CompanyName, Phone, Email *string; IsActive *bool }
type Repository interface {
    List(context.Context, uuid.UUID) ([]Client, error)
    Create(context.Context, *Client) error
    Find(context.Context, uuid.UUID, uuid.UUID) (Client, error)
    Update(context.Context, *Client) error
}
```

Todas las consultas filtran `workshop_id`. Solo OWNER/ADMIN registran y modifican; OPERATOR puede recibir los datos mínimos del cliente únicamente como parte de una orden asignada, no mediante estas rutas.

- [ ] **Step 4: Implementar vista y formulario Vue**

Probar que nombre es obligatorio, email inválido se bloquea, guardar emite el cliente creado y los estados vacío/error usan `ViewState`.

- [ ] **Step 5: Verificar módulo completo**

Run: `cd backend; go test ./internal/clients; cd ../frontend; npm run test -- --run src/modules/clients`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/clients backend/internal/platform/httpx/router.go frontend/src/modules/clients
git commit -m "feat: manage workshop clients"
```

### Task 6: Crear y listar órdenes con sus etapas

**Files:**
- Create: `backend/internal/orders/model.go`
- Create: `backend/internal/orders/dto.go`
- Create: `backend/internal/orders/repository.go`
- Create: `backend/internal/orders/service.go`
- Create: `backend/internal/orders/handler.go`
- Create: `backend/internal/orders/service_test.go`
- Create: `backend/internal/orders/handler_test.go`
- Create: `backend/internal/orders/repository_integration_test.go`
- Modify: `backend/internal/platform/httpx/router.go`

**Interfaces:**
- Consumes: cliente del mismo taller, principal OWNER/ADMIN y transacciones GORM.
- Produces: `orders.Service.Create/List/Get/Update`, `POST /orders`, `GET /orders`, `GET /orders/:id`, `PATCH /orders/:id`; tipos `Order`, `Stage`, `CreateInput`, `UpdateInput`, `ListFilter` y `OrderSummary` usados por producción y dashboard.

- [ ] **Step 1: Escribir pruebas del agregado de creación**

```go
func TestCreateOrderBuildsStandardStages(t *testing.T) {
    in := validCreateInput()
    in.ApplicableStages = []string{"CORTE", "CONFECCION", "ACABADO", "CONTROL_CALIDAD", "ENTREGA"}
    got, _ := service.Create(ctx, owner, in)
    assert.Equal(t, "COMPLETED", got.Stages[0].Status)
    assert.Equal(t, "IN_PROGRESS", got.Stages[1].Status)
    assert.Equal(t, "SKIPPED", got.Stages[3].Status)
}
```

Agregar al mismo archivo esta matriz y materializar cada fila como un test independiente:

```yaml
- test: TestCreateOrderRejectsForeignClient
  given: "client belongs to another workshop"
  expect: "ErrNotFound"
- test: TestConcurrentCodesAreUniqueAndSequential
  given: "two goroutines create orders for the same workshop"
  expect: "codes are TF-0001 and TF-0002, with no duplicate"
- test: TestOperatorCannotCreateOrder
  given: "principal role OPERATOR"
  expect: "403 ROLE_FORBIDDEN"
- test: TestUpdateRejectsCompletedOrder
  given: "order status COMPLETED"
  input: {delivery_date: "2026-10-15"}
  expect: "409 ORDER_NOT_EDITABLE"
```

- [ ] **Step 2: Confirmar el fallo**

Run: `cd backend; go test ./internal/orders`

Expected: FAIL porque el módulo no existe.

- [ ] **Step 3: Implementar creación transaccional**

```go
type CreateInput struct {
    ClientID uuid.UUID
    Product, Description string
    Quantity int
    StartDate, DeliveryDate time.Time
    ResponsibleUserID *uuid.UUID
    ApplicableStages []StageCode
}
```

Bloquear `workshop_counters` con `FOR UPDATE`, formar `TF-%04d`, generar 32 bytes con `crypto/rand`, codificar Base64 URL-safe sin padding, insertar orden y siete etapas, asignar `current_stage_id` y confirmar en una transacción. Validar cantidad, fechas, cliente, responsable y que `ENTREGA` sea aplicable. `PEDIDO_CONFIRMADO` usa `tracks_quantity=false`; las otras etapas aplicables usan `tracks_quantity=true` y `target_quantity=orders.quantity`. Si hay responsable inicial, asignarlo también a la primera etapa productiva.

- [ ] **Step 4: Implementar listado paginado y filtros**

`ListFilter` incluye `Status`, `ClientID`, `Stage`, `DeliveryFrom`, `DeliveryTo`, `Page`, `PageSize`; limitar `PageSize` a 100. La respuesta incluye `{data: [...], meta: {page,page_size,total}}` sin exponer `tracking_token`.

`UpdateInput` admite `product`, `description`, `delivery_date` y `responsible_user_id`. Solo OWNER/ADMIN actualizan órdenes `ACTIVE` o `PAUSED`; cliente, cantidad, fecha de inicio y etapas ya creadas permanecen inmutables.

- [ ] **Step 5: Ejecutar pruebas unitarias, HTTP e integración concurrente**

Run: `cd backend; go test ./internal/orders -race`

Expected: PASS; las consultas ajenas al taller devuelven not found.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/orders backend/internal/platform/httpx/router.go
git commit -m "feat: create and list textile orders"
```

### Task 7: UI de listado y creación de órdenes

**Files:**
- Create: `frontend/src/modules/orders/services/orders.js`
- Create: `frontend/src/modules/orders/views/OrdersView.vue`
- Create: `frontend/src/modules/orders/views/CreateOrderView.vue`
- Create: `frontend/src/modules/orders/components/OrderFilters.vue`
- Create: `frontend/src/modules/orders/components/OrderCard.vue`
- Create: `frontend/src/modules/orders/components/OrderForm.vue`
- Create: `frontend/src/modules/orders/components/OrderForm.test.js`
- Create: `frontend/src/modules/orders/views/OrdersView.test.js`
- Modify: `frontend/src/router/index.js`
- Modify: `frontend/src/layouts/AppLayout.vue`

**Interfaces:**
- Consumes: endpoints de clientes y órdenes de las tareas 5–6.
- Produces: rutas `/orders` y `/orders/new`, filtros reflejados en query string y navegación al detalle `/orders/:id`.

- [ ] **Step 1: Escribir pruebas del formulario y listado**

```yaml
- test: "no envía una orden con entrega anterior al inicio"
  input: {start_date: "2026-09-22", delivery_date: "2026-09-21"}
  expect: "orders.create is not called; delivery_date inline error is visible"
- test: "envía solo etapas seleccionadas y cantidad numérica"
  input: {quantity: "100", stages: [CORTE, CONFECCION, ENTREGA]}
  expect: "orders.create receives quantity=100 and the exact stage array"
- test: "sincroniza status y page con route.query"
  route_query: {status: ACTIVE, page: "2"}
  expect: "orders.list receives status=ACTIVE and page=2"
```

- [ ] **Step 2: Confirmar el fallo**

Run: `cd frontend; npm run test -- --run src/modules/orders`

Expected: FAIL porque las vistas no existen.

- [ ] **Step 3: Implementar listado responsive**

En desktop usar tabla; en móvil tarjetas. Mostrar código, cliente, producto, etapa actual, progreso, entrega y semáforo. Incluir carga, vacío, sin resultados y error recuperable.

- [ ] **Step 4: Implementar creación guiada**

El formulario selecciona/crea cliente, captura producto, cantidad, fechas, responsable y etapas. `Pedido confirmado` aparece fijo; las etapas opcionales pueden desmarcarse; `Entrega` permanece obligatoria. Tras `201`, navegar a `/orders/{id}`.

- [ ] **Step 5: Ejecutar pruebas y build**

Run: `cd frontend; npm run test -- --run src/modules/orders; npm run build`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/modules/orders frontend/src/router/index.js frontend/src/layouts/AppLayout.vue
git commit -m "feat: add order list and creation flow"
```

## Fase 3 — Trazabilidad productiva

### Task 8: Detalle y transición segura de etapas

**Files:**
- Create: `backend/internal/production/model.go`
- Create: `backend/internal/production/dto.go`
- Create: `backend/internal/production/repository.go`
- Create: `backend/internal/production/service.go`
- Create: `backend/internal/production/handler.go`
- Create: `backend/internal/production/stages_test.go`
- Create: `backend/internal/production/stages_integration_test.go`
- Modify: `backend/internal/platform/httpx/router.go`

**Interfaces:**
- Consumes: `orders.Order`, `orders.Stage`, `auth.Principal`.
- Produces: `production.Service.Detail/Assign/Advance`, `GET /orders/:id/stages`, `PATCH /orders/:id/stages/:stageId/assignee`, `POST /orders/:id/stages/:stageId/advance`.

- [ ] **Step 1: Escribir pruebas de transición y concurrencia**

```yaml
- test: TestAdvanceRequiresCurrentStage
  given: "requested stage differs from order.current_stage_id"
  expect: "409 STAGE_NOT_CURRENT"
- test: TestAdvanceRequiresTargetQuantity
  given: "tracks_quantity=true, accumulated=99, target=100"
  expect: "422 STAGE_TARGET_NOT_REACHED"
- test: TestAdvanceSkipsSkippedStages
  given: "current CORTE, next ESTAMPADO is SKIPPED, then ACABADO is PENDING"
  expect: "CORTE COMPLETED and ACABADO IN_PROGRESS"
- test: TestAdvanceLastStageCompletesOrder
  given: "ENTREGA is current and its target is reached"
  expect: "order COMPLETED with current_stage_id null"
- test: TestConcurrentAdvanceAllowsOnlyOneCommit
  given: "two transactions advance the same current stage"
  expect: "one success, one 409, exactly one STAGE_ADVANCED event"
- test: TestOverallProgressExcludesSkippedStages
  given: "four applicable stages: two completed, current quantitative stage at 50%, one pending; three skipped"
  expect: "progress_percent=62.5"
```

- [ ] **Step 2: Ejecutar pruebas y confirmar el fallo**

Run: `cd backend; go test ./internal/production -run 'TestAdvance|TestConcurrentAdvance'`

Expected: FAIL porque el servicio no existe.

- [ ] **Step 3: Implementar detalle y asignación**

`Detail` devuelve resumen, las siete etapas, acumulado por etapa e historial paginado. El progreso general es el promedio de las etapas no `SKIPPED`: `COMPLETED=100`, `PENDING=0`, etapa cuantitativa actual=`acumulado/objetivo*100` y etapa actual no cuantitativa=`0` hasta completarse. `Assign` verifica que el usuario sea una membresía activa del mismo taller. OWNER/ADMIN pueden asignar; OPERATOR solo consulta órdenes asignadas.

- [ ] **Step 4: Implementar avance con bloqueo pesimista**

Dentro de la transacción bloquear orden y etapas actuales con `clause.Locking{Strength: "UPDATE"}`, volver a validar estado y etapa, completar la actual, saltar `SKIPPED`, activar la siguiente o completar la orden e insertar `STAGE_ADVANCED`/`ORDER_COMPLETED`.

- [ ] **Step 5: Ejecutar pruebas con detector de carreras**

Run: `cd backend; go test ./internal/production -race`

Expected: PASS; la prueba concurrente observa un único avance.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/production backend/internal/platform/httpx/router.go
git commit -m "feat: expose order detail and stage transitions"
```

### Task 9: Avances incrementales, idempotencia y correcciones

**Files:**
- Create: `backend/internal/production/updates.go`
- Create: `backend/internal/production/updates_test.go`
- Create: `backend/internal/production/updates_integration_test.go`
- Modify: `backend/internal/production/dto.go`
- Modify: `backend/internal/production/repository.go`
- Modify: `backend/internal/production/service.go`
- Modify: `backend/internal/production/handler.go`

**Interfaces:**
- Consumes: etapa actual bloqueada, asignación del operario e índice `ux_order_updates_request`.
- Produces: `RecordProgress`, `CorrectProgress`, `ListUpdates`, `GET|POST /orders/:id/updates` y `POST /orders/:id/updates/:updateId/corrections`.

- [ ] **Step 1: Escribir pruebas del libro append-only**

```yaml
- test: TestRecordProgressAddsDelta
  given: "existing accumulated=20 and target=100"
  input: {quantity_delta: 15}
  expect: "accumulated=35"
- test: TestRecordProgressRejectsAboveTarget
  given: "existing accumulated=95 and target=100"
  input: {quantity_delta: 6}
  expect: "422 PROGRESS_EXCEEDS_TARGET"
- test: TestOperatorMustBeAssigned
  given: "principal is OPERATOR B but current stage is assigned to OPERATOR A"
  expect: "404 ORDER_NOT_FOUND"
- test: TestCorrectionAppendsNegativeDelta
  given: "original quantity_delta=20"
  input: {quantity_delta: -5}
  expect: "original remains 20; correction references original; accumulated decreases by 5"
- test: TestConcurrentDuplicateClientRequestCreatesOneUpdate
  given: "two transactions use the same non-null client_request_id"
  expect: "both responses identify the same update and database count is 1"
```

- [ ] **Step 2: Confirmar el fallo**

Run: `cd backend; go test ./internal/production -run 'TestRecord|TestCorrection|TestConcurrentDuplicate'`

Expected: FAIL.

- [ ] **Step 3: Implementar comandos exactos**

```go
type RecordProgressInput struct {
    StageID uuid.UUID `json:"stage_id"`
    QuantityDelta int `json:"quantity_delta"`
    Comment string `json:"comment"`
    ClientRequestID uuid.UUID `json:"client_request_id"`
}
type CorrectProgressInput struct {
    QuantityDelta int `json:"quantity_delta"`
    Comment string `json:"comment"`
    ClientRequestID uuid.UUID `json:"client_request_id"`
}
```

En una transacción: buscar una respuesta existente por `(order_id, client_request_id)`; si existe devolverla; bloquear orden/etapa; sumar deltas; validar `0 <= acumulado <= objetivo`; insertar. Solo OWNER/ADMIN corrigen y la corrección referencia `corrects_update_id`.

- [ ] **Step 4: Mapear colisión del índice a respuesta idempotente**

Si dos transacciones compiten y una recibe unique violation `23505`, releer el evento ganador y responder `200` con el mismo DTO; otros conflictos de estado responden `409`.

- [ ] **Step 5: Ejecutar pruebas completas del módulo**

Run: `cd backend; go test ./internal/production -race`

Expected: PASS y un solo registro para el request duplicado.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/production
git commit -m "feat: record idempotent production progress"
```

### Task 10: Pausa, reanudación, reporte de problema y cancelación

**Files:**
- Create: `backend/internal/production/lifecycle.go`
- Create: `backend/internal/production/lifecycle_test.go`
- Create: `backend/internal/production/lifecycle_integration_test.go`
- Modify: `backend/internal/production/service.go`
- Modify: `backend/internal/production/handler.go`
- Modify: `backend/internal/platform/httpx/router.go`

**Interfaces:**
- Consumes: orden y etapa actual con bloqueo.
- Produces: `Pause`, `Resume`, `ReportProblem`, `Cancel`; rutas `POST /orders/:id/pause`, `/resume`, `/cancel` y un `POST /orders/:id/updates` con `event_type=PROBLEM_REPORTED` para el operario.

- [ ] **Step 1: Escribir la matriz de transiciones**

```yaml
- test: TestPauseBlocksCurrentStage
  given: "order ACTIVE and stage IN_PROGRESS"
  expect: "order PAUSED, stage BLOCKED, one ORDER_PAUSED event"
- test: TestResumeRestoresCurrentStage
  given: "order PAUSED and stage BLOCKED"
  expect: "order ACTIVE, stage IN_PROGRESS, one ORDER_RESUMED event"
- test: TestOperatorReportsProblemWithoutPausing
  given: "assigned OPERATOR and ACTIVE order"
  expect: "one PROBLEM_REPORTED event and order remains ACTIVE"
- test: TestCancelClearsCurrentStageAndDisablesTracking
  given: "ACTIVE order with tracking enabled"
  expect: "CANCELLED, current_stage_id null, tracking_enabled false"
- test: TestConcurrentPauseAndCancelReturnsOneConflict
  given: "pause and cancel start against the same ACTIVE order"
  expect: "one commits; the stale operation returns 409; no partial stage state"
```

- [ ] **Step 2: Confirmar el fallo**

Run: `cd backend; go test ./internal/production -run 'TestPause|TestResume|TestOperatorReports|TestCancel|TestConcurrentPause'`

Expected: FAIL.

- [ ] **Step 3: Implementar la máquina de estados**

Aceptar `reason` obligatorio para pausa/cancelación y `comment` opcional. OWNER/ADMIN pausan, reanudan y cancelan. OPERATOR asignado solo crea `PROBLEM_REPORTED`. Cada operación cambia estado e inserta evento dentro de la misma transacción.

- [ ] **Step 4: Ejecutar pruebas unitarias e integración concurrente**

Run: `cd backend; go test ./internal/production -race`

Expected: PASS; ningún estado parcial queda persistido.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/production backend/internal/platform/httpx/router.go
git commit -m "feat: manage order production lifecycle"
```

### Task 11: Evidencias seguras mediante Supabase Storage

**Files:**
- Create: `backend/internal/attachments/model.go`
- Create: `backend/internal/attachments/storage.go`
- Create: `backend/internal/attachments/supabase_storage.go`
- Create: `backend/internal/attachments/service.go`
- Create: `backend/internal/attachments/handler.go`
- Create: `backend/internal/attachments/service_test.go`
- Create: `backend/internal/attachments/handler_test.go`
- Modify: `backend/internal/platform/httpx/router.go`

**Interfaces:**
- Consumes: `Storage.Put(ctx, key, mime, reader, size) error`, `Storage.SignedURL(ctx, key, ttl) (string, error)` y update perteneciente a la orden/taller.
- Produces: `POST /orders/:id/updates/:updateId/attachments`, `PATCH /orders/:id/attachments/:attachmentId/visibility` y DTO con URL temporal autorizada.

- [ ] **Step 1: Escribir pruebas de seguridad de archivos**

```yaml
- test: TestUploadRejectsMoreThan10MB
  given: "multipart image body has 10MB + 1 byte"
  expect: "413 ATTACHMENT_TOO_LARGE and Storage.Put not called"
- test: TestUploadRejectsFakeJPEG
  given: "filename photo.jpg with text/plain bytes"
  expect: "422 ATTACHMENT_INVALID_IMAGE"
- test: TestUploadAcceptsDecodedPNGAsInternal
  given: "valid 1x1 PNG"
  expect: "201 with is_client_visible=false"
- test: TestOperatorCannotPublishAttachment
  given: "role OPERATOR and existing attachment"
  expect: "403 ROLE_FORBIDDEN"
- test: TestStorageFailureDoesNotDeleteProgress
  given: "existing progress update and Storage.Put returns ErrUnavailable"
  expect: "502 STORAGE_UNAVAILABLE and progress update count remains 1"
```

- [ ] **Step 2: Confirmar el fallo**

Run: `cd backend; go test ./internal/attachments`

Expected: FAIL.

- [ ] **Step 3: Implementar validación y persistencia**

Limitar el body con `http.MaxBytesReader`, detectar MIME con `http.DetectContentType`, decodificar con `image.DecodeConfig`, registrar WebP mediante import de `golang.org/x/image/webp`, permitir únicamente JPEG/PNG/WebP y generar `{workshop_id}/{order_id}/{uuid}` sin reutilizar el nombre recibido. Subir primero y persistir metadata después; si la inserción falla, borrar el objeto mediante `Storage.Delete`.

- [ ] **Step 4: Implementar adaptador Supabase y visibilidad**

Usar HTTP autenticado con service-role solo desde backend. Para archivos internos o públicos generar URL firmada de corta duración; nunca exponer la service-role. `visibility` solo acepta `{is_client_visible: boolean}` para OWNER/ADMIN.

- [ ] **Step 5: Ejecutar pruebas**

Run: `cd backend; go test ./internal/attachments`

Expected: PASS, incluyendo MIME falso y fallo remoto.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/attachments backend/internal/platform/httpx/router.go
git commit -m "feat: attach secure production evidence"
```

### Task 12: Experiencia de detalle para dueño y operario

**Files:**
- Create: `frontend/src/modules/production/services/production.js`
- Create: `frontend/src/modules/production/views/OrderDetailView.vue`
- Create: `frontend/src/modules/production/views/OperatorOrdersView.vue`
- Create: `frontend/src/modules/production/components/StageTimeline.vue`
- Create: `frontend/src/modules/production/components/ProgressForm.vue`
- Create: `frontend/src/modules/production/components/OrderActions.vue`
- Create: `frontend/src/modules/production/components/UpdateHistory.vue`
- Create: `frontend/src/modules/production/components/AttachmentUploader.vue`
- Create: `frontend/src/modules/production/components/ProgressForm.test.js`
- Create: `frontend/src/modules/production/views/OrderDetailView.test.js`
- Modify: `frontend/src/router/index.js`

**Interfaces:**
- Consumes: endpoints de tareas 8–11 y rol de la store de sesión.
- Produces: `/orders/:id`, `/my-orders`, registro móvil de avance, acciones autorizadas y carga/publicación de evidencia.

- [ ] **Step 1: Escribir pruebas de interacción y permisos**

```yaml
- test: "genera client_request_id una vez y lo conserva al reintentar"
  given: "first request fails with a network error"
  expect: "second submit sends the same UUID"
- test: "oculta acciones administrativas al operador"
  given: "session role OPERATOR"
  expect: "correct, advance, publish, pause and cancel controls are absent"
- test: "muestra conflicto y recarga detalle"
  given: "mutation responds 409"
  expect: "conflict message visible and production.detail called again"
- test: "conserva avance visible cuando falla fotografía"
  given: "progress succeeds and attachment upload fails"
  expect: "new update remains visible and retry-photo action appears"
```

- [ ] **Step 2: Confirmar el fallo**

Run: `cd frontend; npm run test -- --run src/modules/production`

Expected: FAIL.

- [ ] **Step 3: Implementar detalle de OWNER/ADMIN**

Mostrar encabezado, semáforo, progreso, timeline, responsable, historial, evidencias y modales de corregir, pausar, reanudar, cancelar y avanzar. Tras mutaciones invalidar y releer el detalle; no actualizar optimistamente el historial.

- [ ] **Step 4: Implementar captura mobile-first de OPERATOR**

`ProgressForm` ofrece accesos `+1`, `+5`, `+10`, cantidad manual, comentario, foto opcional y reporte de problema. Deshabilitar doble envío, mantener el UUID durante reintentos y anunciar éxito/error con región `aria-live`.

- [ ] **Step 5: Ejecutar pruebas y build**

Run: `cd frontend; npm run test -- --run src/modules/production; npm run build`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/modules/production frontend/src/router/index.js
git commit -m "feat: add owner and operator production views"
```

## Fase 4 — Visibilidad y administración

### Task 13: Dashboard y semáforo de entrega

**Files:**
- Create: `backend/internal/dashboard/dto.go`
- Create: `backend/internal/dashboard/repository.go`
- Create: `backend/internal/dashboard/service.go`
- Create: `backend/internal/dashboard/handler.go`
- Create: `backend/internal/dashboard/service_test.go`
- Modify: `backend/internal/platform/httpx/router.go`
- Create: `frontend/src/modules/dashboard/services/dashboard.js`
- Create: `frontend/src/modules/dashboard/views/DashboardView.vue`
- Create: `frontend/src/modules/dashboard/views/DashboardView.test.js`
- Modify: `frontend/src/router/index.js`

**Interfaces:**
- Consumes: órdenes y etapas del taller; reloj inyectable `Clock.Now() time.Time`.
- Produces: `DeliveryHealth(order, nowInLima)`, `GET /dashboard/summary` y tablero inicial.

- [ ] **Step 1: Escribir tabla de pruebas del semáforo, incluida medianoche UTC**

```go
func TestDeliveryHealthUsesLimaDate(t *testing.T) {
    now := time.Date(2026, 9, 22, 4, 30, 0, 0, time.UTC) // 21/09 23:30 en Lima
    assert.Equal(t, OnTime, DeliveryHealth(orderDue("2026-09-22", "CORTE"), now))
}
// Casos adicionales: vencida=>LATE, hoy/mañana fuera de última etapa=>AT_RISK,
// completada=>COMPLETED, mañana en última etapa=>ON_TIME.
```

- [ ] **Step 2: Confirmar el fallo**

Run: `cd backend; go test ./internal/dashboard`

Expected: FAIL.

- [ ] **Step 3: Implementar resumen tenant-aware**

El DTO incluye conteos `active`, `paused`, `late`, `at_risk`, `completed_this_month`, próximas entregas y órdenes con problemas recientes. Convertir el reloj a `America/Lima` antes de comparar con `DATE`.

- [ ] **Step 4: Implementar DashboardView**

Crear tarjetas, lista de próximas entregas y accesos al detalle; contemplar carga, ausencia de órdenes y error con reintento.

- [ ] **Step 5: Verificar ambas capas**

Run: `cd backend; go test ./internal/dashboard; cd ../frontend; npm run test -- --run src/modules/dashboard`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/dashboard backend/internal/platform/httpx/router.go frontend/src/modules/dashboard frontend/src/router/index.js
git commit -m "feat: add operational dashboard"
```

### Task 14: Tracking público y gestión del token

**Files:**
- Create: `backend/internal/tracking/dto.go`
- Create: `backend/internal/tracking/repository.go`
- Create: `backend/internal/tracking/service.go`
- Create: `backend/internal/tracking/handler.go`
- Create: `backend/internal/tracking/service_test.go`
- Create: `backend/internal/tracking/handler_test.go`
- Modify: `backend/internal/platform/httpx/router.go`
- Create: `frontend/src/modules/tracking/services/tracking.js`
- Create: `frontend/src/modules/tracking/views/TrackingView.vue`
- Create: `frontend/src/modules/tracking/views/TrackingView.test.js`
- Modify: `frontend/src/router/index.js`

**Interfaces:**
- Consumes: orden, etapas, updates y attachments publicados.
- Produces: `GET /tracking/:token` sin JWT, `PATCH /orders/:id/tracking`, `POST /orders/:id/tracking/regenerate` y `/tracking/:token` en Vue.

- [ ] **Step 1: Escribir pruebas de privacidad**

```yaml
- test: TestPublicDTOOmitsInternalFields
  expect_absent_keys: [user_id, assigned_user_id, tracking_token, internal_comment, workshop_id]
- test: TestTrackingOnlyIncludesVisibleAttachments
  given: "one visible and one internal attachment"
  expect: "response contains only the visible attachment"
- test: TestDisabledOrCancelledTrackingReturnsNotFound
  cases: ["tracking_enabled=false", "order.status=CANCELLED"]
  expect: "404 TRACKING_NOT_FOUND"
- test: TestRegenerateImmediatelyInvalidatesOldToken
  given: "valid old token"
  expect: "old token returns 404 and newly returned URL returns 200"
```

- [ ] **Step 2: Confirmar el fallo**

Run: `cd backend; go test ./internal/tracking`

Expected: FAIL.

- [ ] **Step 3: Implementar proyección pública**

Exponer código, producto, cantidad, nombre comercial del taller, estado, etapa actual, progreso general, fechas, timeline sanitizado, última actualización, evidencias visibles y teléfono del taller. Omitir emails, UUID internos, asignados, comentarios internos y el propio token.

- [ ] **Step 4: Implementar revocación/regeneración y vista externa**

Solo OWNER/ADMIN cambian `tracking_enabled` o generan 32 bytes nuevos dentro de una transacción. `TrackingView` no carga shell interno, ofrece estados inválido/no disponible y presenta una línea de tiempo simple responsive.

- [ ] **Step 5: Verificar API y UI**

Run: `cd backend; go test ./internal/tracking; cd ../frontend; npm run test -- --run src/modules/tracking`

Expected: PASS y el JSON público no contiene claves prohibidas.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/tracking backend/internal/platform/httpx/router.go frontend/src/modules/tracking frontend/src/router/index.js
git commit -m "feat: publish private customer tracking"
```

### Task 15: Equipo, invitaciones y asignaciones válidas

**Files:**
- Create: `backend/internal/team/model.go`
- Create: `backend/internal/team/inviter.go`
- Create: `backend/internal/team/supabase_inviter.go`
- Create: `backend/internal/team/repository.go`
- Create: `backend/internal/team/service.go`
- Create: `backend/internal/team/handler.go`
- Create: `backend/internal/team/service_test.go`
- Create: `backend/internal/team/handler_test.go`
- Modify: `backend/internal/auth/service.go`
- Modify: `backend/internal/platform/httpx/router.go`
- Create: `frontend/src/modules/team/services/team.js`
- Create: `frontend/src/modules/team/views/TeamView.vue`
- Create: `frontend/src/modules/team/components/InviteMemberForm.vue`
- Create: `frontend/src/modules/team/components/InviteMemberForm.test.js`
- Modify: `frontend/src/router/index.js`

**Interfaces:**
- Consumes: Supabase Admin invite detrás de `Inviter.Invite(ctx, email, redirectURL) error` y aceptación durante `ResolvePrincipal`.
- Produces: `GET /team`, `POST /team/invitations`, `PATCH /team/:membershipId`, membresías activas y UI de equipo.

- [ ] **Step 1: Escribir pruebas de privilegios e invitación**

```yaml
- test: TestInviteCreatesPendingInvitationAfterSupabaseAccepts
  given: "Inviter.Invite succeeds"
  expect: "one invitation with status PENDING and configured expires_at"
- test: TestAdminCannotAssignOwnerRole
  given: "actor ADMIN"
  input: {role: OWNER}
  expect: "403 ROLE_FORBIDDEN"
- test: TestCannotDeactivateLastOwner
  given: "target is the only active OWNER"
  input: {status: INACTIVE}
  expect: "422 LAST_OWNER_REQUIRED"
- test: TestFirstLoginConsumesValidInvitation
  given: "unexpired PENDING invitation matching JWT email"
  expect: "user and membership exist; invitation ACCEPTED in one transaction"
- test: TestExpiredInvitationIsNotConsumed
  given: "expires_at is before clock.Now"
  expect: "403 MEMBERSHIP_REQUIRED and invitation remains unaccepted"
```

- [ ] **Step 2: Confirmar el fallo**

Run: `cd backend; go test ./internal/team ./internal/auth`

Expected: FAIL.

- [ ] **Step 3: Implementar administración de equipo**

OWNER y ADMIN administran membresías `ADMIN` y `OPERATOR`; solo OWNER puede otorgar o modificar el rol `OWNER`, y nadie puede desactivar al último OWNER. Normalizar email en minúsculas, impedir invitaciones pendientes duplicadas y usar expiración explícita.

- [ ] **Step 4: Consumir invitación durante el primer acceso**

Dentro de una transacción, bloquear invitación pendiente por email, comprobar expiración, crear/actualizar usuario, crear membresía y marcar `ACCEPTED`. Si no existe membresía ni invitación válida, negar acceso aunque el JWT sea auténtico.

- [ ] **Step 5: Implementar TeamView y verificar**

Mostrar miembros y estado, invitar con rol permitido, activar/desactivar membresías y ocultar controles no autorizados. Ejecutar: `cd backend; go test ./internal/team ./internal/auth; cd ../frontend; npm run test -- --run src/modules/team`.

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/team backend/internal/auth backend/internal/platform/httpx/router.go frontend/src/modules/team frontend/src/router/index.js
git commit -m "feat: manage workshop team invitations"
```

## Fase 5 — Endurecimiento y entrega

### Task 16: Errores uniformes, logs, CORS y rate limiting

**Files:**
- Create: `backend/internal/platform/httpx/errors.go`
- Create: `backend/internal/platform/httpx/errors_test.go`
- Create: `backend/internal/platform/httpx/request_id.go`
- Create: `backend/internal/platform/httpx/logger.go`
- Create: `backend/internal/platform/httpx/cors.go`
- Create: `backend/internal/platform/httpx/ratelimit.go`
- Create: `backend/internal/platform/httpx/security_test.go`
- Modify: `backend/internal/platform/httpx/router.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: errores de dominio de todos los módulos.
- Produces: contrato único `{error:{code,message,details,request_id}}`, logs JSON y límites por IP para tracking/cargas.

- [ ] **Step 1: Escribir pruebas del perímetro HTTP**

```yaml
- test: TestForeignResourceMapsTo404WithoutDetails
  given: "domain ErrNotFound from tenant-filtered repository"
  expect: "404 with empty details"
- test: TestPanicReturns500WithRequestID
  given: "handler panics after request ID middleware"
  expect: "500 INTERNAL_ERROR with the same X-Request-ID; next request still succeeds"
- test: TestLogsRedactAuthorizationAndTrackingToken
  given: "Bearer secret and GET /tracking/secret-token"
  expect: "captured JSON log contains neither secret string"
- test: TestCORSRejectsUnknownOrigin
  given: "Origin https://attacker.example"
  expect: "response has no Access-Control-Allow-Origin"
- test: TestTrackingRateLimitReturns429
  given: "same IP exceeds configured requests per minute"
  expect: "429 RATE_LIMITED with Retry-After"
```

- [ ] **Step 2: Ejecutar y confirmar el fallo**

Run: `cd backend; go test ./internal/platform/httpx`

Expected: FAIL.

- [ ] **Step 3: Implementar middleware en orden determinista**

Orden: request ID, recovery, logger, CORS, límite público, autenticación para grupos privados y handlers. Registrar método, plantilla de ruta, status, duración, user/workshop cuando existan y `error_code`; nunca registrar query completa de tracking, bearer token, body o nombres de archivo.

- [ ] **Step 4: Normalizar errores en todos los handlers**

Mapear validación a `400`, auth a `401`, rol a `403`, not found/foreign a `404`, conflicto a `409`, regla de negocio a `422`, tamaño a `413` y desconocido a `500`. Incluir detalles solo para validaciones de campos.

- [ ] **Step 5: Verificar todo el backend**

Run: `cd backend; go test ./... -race; go vet ./...`

Expected: PASS sin secretos capturados por las pruebas.

- [ ] **Step 6: Commit**

```bash
git add backend
git commit -m "feat: harden TallerFlow HTTP boundary"
```

### Task 17: Pruebas end-to-end del recorrido crítico

**Files:**
- Create: `frontend/playwright.config.js`
- Create: `frontend/e2e/fixtures/auth.js`
- Create: `frontend/e2e/order-traceability.spec.js`
- Create: `frontend/e2e/role-permissions.spec.js`
- Create: `frontend/e2e/tracking.spec.js`
- Modify: `frontend/package.json`
- Create: `backend/cmd/e2e-seed/main.go`
- Modify: `compose.yaml`

**Interfaces:**
- Consumes: stack completa, un proyecto Supabase dedicado a pruebas y usuarios E2E OWNER/OPERATOR provisionados solo en perfil `e2e`.
- Produces: comandos `npm run test:e2e` y `docker compose --profile e2e up` para validar el MVP desde navegador.

- [ ] **Step 1: Escribir primero el recorrido que falla**

```js
test('owner creates order, operator advances it, client sees tracking', async ({ page }) => {
  await loginAs(page, 'owner')
  await createClient(page, 'Textiles Demo')
  const { code, trackingUrl } = await createOrder(page, { quantity: 100 })
  await loginAs(page, 'operator')
  await recordProgress(page, code, 100)
  await loginAs(page, 'owner')
  await advanceCurrentStage(page, code)
  await page.goto(trackingUrl)
  await expect(page.getByText(code)).toBeVisible()
  await expect(page.getByText('Confección')).toBeVisible()
})
```

- [ ] **Step 2: Ejecutar y confirmar el fallo de infraestructura E2E**

Run: `cd frontend; npm run test:e2e`

Expected: FAIL porque servidor/fixtures E2E aún no están configurados.

- [ ] **Step 3: Configurar fixtures deterministas**

Usar un proyecto Supabase dedicado a pruebas configurado con `E2E_SUPABASE_URL`, `E2E_SUPABASE_ANON_KEY` y `E2E_SUPABASE_SERVICE_ROLE_KEY`; el comando debe rechazar un host que coincida con `SUPABASE_PRODUCTION_URL`. `backend/cmd/e2e-seed` crea o restablece OWNER/OPERATOR mediante Supabase Admin, toma sus `auth_user_id` reales y reemplaza solo los datos del taller cuyo nombre es `TallerFlow E2E`. Cada spec crea sus propias órdenes con UUID distintos.

- [ ] **Step 4: Completar tres recorridos**

1. Flujo crítico completo de orden y tracking.
2. OPERATOR no ve equipo, corrección, publicación ni cancelación; tampoco puede invocar esas rutas directamente.
3. Token regenerado invalida la URL anterior y una evidencia interna no aparece públicamente.

- [ ] **Step 5: Ejecutar en Chromium desktop y viewport móvil**

Run: `cd frontend; npm run test:e2e`

Expected: los tres specs pasan en ambos proyectos; Playwright conserva traza únicamente al reintentar.

- [ ] **Step 6: Commit**

```bash
git add frontend backend/cmd/e2e-seed compose.yaml
git commit -m "test: cover TallerFlow critical journeys"
```

### Task 18: CI, documentación operativa y verificación final

**Files:**
- Create: `.github/workflows/ci.yml`
- Modify: `README.md`
- Create: `docs/operations/deployment.md`
- Create: `docs/operations/runbook.md`

**Interfaces:**
- Consumes: todos los comandos de comprobación anteriores.
- Produces: pipeline reproducible, guía de entorno local, contrato de migración previa y lista de verificación de despliegue.

- [ ] **Step 1: Crear CI con servicios reales**

```yaml
jobs:
  backend:
    services:
      postgres:
        image: postgres:17
    steps:
      - run: flyway validate
      - run: flyway migrate
      - run: cd backend && go test ./... -race && go vet ./...
  frontend:
    steps:
      - run: cd frontend && npm ci
      - run: cd frontend && npm run lint && npm run test -- --run && npm run build
```

Usar `actions/checkout@v4`, `actions/setup-go@v5` con la versión declarada en `backend/go.mod` y `actions/setup-node@v4` con caché npm y Node 22. El servicio PostgreSQL define `POSTGRES_DB=tallerflow_test`, `POSTGRES_USER=tallerflow`, `POSTGRES_PASSWORD=tallerflow`, publica `5432:5432` y usa `pg_isready` como health check. El job backend define `TEST_DATABASE_URL=postgres://tallerflow:tallerflow@localhost:5432/tallerflow_test?sslmode=disable` y ejecuta Flyway mediante la misma imagen fijada en `compose.yaml`. El job E2E arranca Compose, espera hasta diez intentos de cinco segundos por `/health/ready` y luego ejecuta Playwright.

- [ ] **Step 2: Documentar operación exacta**

README incluye requisitos, copia de `.env.example`, `docker compose up --build`, bootstrap del primer OWNER, URLs locales, comandos de prueba y solución de los cinco errores comunes. `deployment.md` exige backup, `flyway validate`, `flyway migrate`, arranque de API y smoke tests. `runbook.md` explica investigar `request_id`, fallo de Storage, JWT inválido, migración fallida y conflicto `409`.

- [ ] **Step 3: Ejecutar la puerta completa desde un clon limpio o worktree limpio**

Run:

```powershell
docker compose down -v
docker compose up -d postgres
docker compose run --rm flyway validate
docker compose run --rm flyway migrate
cd backend; go test ./... -race; go vet ./...; cd ..
cd frontend; npm ci; npm run lint; npm run test -- --run; npm run build; npm run test:e2e; cd ..
```

Expected: todos los comandos terminan con código 0 y el recorrido crítico pasa en desktop y móvil.

- [ ] **Step 4: Comprobar criterios arquitectónicos manualmente**

Verificar que no exista `AutoMigrate`, que el frontend no llame PostgreSQL/Supabase Storage directamente, que los logs no muestren tokens, que Flyway migre una base vacía, que otro taller reciba `404` y que el tracking solo contenga la proyección pública.

- [ ] **Step 5: Commit final de entrega**

```bash
git add .github README.md docs/operations
git commit -m "docs: add CI and MVP operations guide"
```

## Hitos de aceptación por fase

1. **Fase 1 — Base ejecutable:** `docker compose up` levanta frontend, API y PostgreSQL; Flyway aplica V1–V6; health checks y builds pasan.
2. **Fase 2 — Primer corte vertical:** OWNER inicia sesión, crea un cliente y una orden con etapas; el listado queda aislado por taller.
3. **Fase 3 — Trazabilidad productiva:** OPERATOR registra avances idempotentes; OWNER controla etapas, estados y evidencias sin perder historial.
4. **Fase 4 — Visibilidad y administración:** dashboard operativo, tracking privado y gestión de equipo funcionan con sus permisos.
5. **Fase 5 — Cierre:** seguridad del perímetro, concurrencia, CI, E2E, operación local y despliegue quedan verificados.

## Orden de entrega recomendado

No abrir la siguiente fase hasta que el hito anterior pase sus pruebas. Dentro de cada fase, ejecutar las tareas en orden porque las interfaces producidas por una tarea son consumidas por la siguiente. Si un cambio de alcance aparece durante la ejecución, actualizar primero la especificación y después este plan antes de modificar código.
