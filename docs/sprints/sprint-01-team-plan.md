# TallerFlow Sprint 1 Team Implementation Plan (OBSOLETO)

> **No iniciar estas tareas.** La arquitectura cambió a autenticación propia en Go, Vue 3 con TypeScript y despliegue en VPS. El Sprint 1 se regenerará tras aprobar la nueva especificación.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Entregar en diez días hábiles una versión local integrada de TallerFlow donde un OWNER pueda iniciar sesión, crear un cliente, crear una orden con sus etapas y consultar el listado de órdenes.

**Architecture:** El sprint construye la base del monolito modular Go, el esquema PostgreSQL administrado por Flyway y el shell Vue 3. El equipo trabajará en cuatro ramas personales desde `main`, integrará mediante pull requests pequeños y respetará propietarios explícitos para los archivos compartidos.

**Tech Stack:** Go, Gin, GORM, PostgreSQL, Flyway, Vue 3, Vite, JavaScript, Pinia, Vue Router, Axios, Tailwind CSS, Vitest, Vue Test Utils, Docker Compose, Supabase Auth y GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-09-21-tallerflow-mvp-design.md`

**Plan maestro:** `docs/superpowers/plans/2026-09-21-tallerflow-mvp-implementation.md`

## Global Constraints

- Flyway es la única autoridad del esquema; GORM no ejecuta AutoMigrate.
- Toda lectura o escritura de negocio está limitada por `workshop_id`.
- Supabase Auth emite la identidad; Go valida el JWT y aplica membresía y rol.
- El frontend usa exclusivamente `/api/v1` para datos de negocio.
- Las cuatro ramas parten de `main`, se integran por pull request y nunca reciben secretos reales.
- El Sprint 1 termina en creación/listado de órdenes; producción, evidencias y tracking quedan fuera.

## Review Focus

- UUID de un cliente u orden perteneciente a otro taller: Michael debe demostrar `404` en pruebas de integración.
- Dos órdenes concurrentes en el mismo taller: Michael debe demostrar códigos secuenciales únicos.
- JWT expirado o membresía inactiva: José debe demostrar rechazo antes de ejecutar el handler.
- Base PostgreSQL totalmente vacía: Stefano debe demostrar `validate → migrate → migrate` sin divergencia.
- Payload frontend distinto al contrato congelado: Lucero debe fijar los cuerpos exactos en pruebas de servicios y formularios.

---

## 1. Información del sprint

| Campo | Definición |
|---|---|
| Sprint | Sprint 1 — Base ejecutable y primera orden |
| Duración asumida | 10 días hábiles |
| Equipo | José, Lucero, Michael y Stefano |
| Rama de integración | `main` |
| Estrategia | Una rama personal por integrante, pull requests por entregable |
| Demostración final | Login → cliente → orden → listado |
| Resultado esperado | Aplicación reproducible con Docker, migraciones validadas y pruebas automáticas |

La duración es una cadencia de trabajo, no una fecha rígida. Si el equipo usa un sprint de duración diferente, debe conservar el mismo orden de dependencias y reducir alcance antes de sacrificar pruebas o aislamiento multi-tenant.

## 2. Alcance comprometido

Al terminar el sprint deben funcionar estos recorridos:

1. Levantar PostgreSQL, Flyway, backend y frontend con Docker Compose.
2. Aplicar las migraciones V1–V6 sobre una base vacía.
3. Consultar `/health/live` y `/health/ready`.
4. Iniciar sesión con Supabase Auth.
5. Resolver el usuario, taller, membresía y rol en la API.
6. Consultar el taller actual.
7. Crear, listar, editar y desactivar clientes.
8. Crear una orden con código secuencial por taller.
9. Crear las siete etapas estándar y marcar las no aplicables como `SKIPPED`.
10. Listar y filtrar órdenes desde una interfaz responsive.
11. Impedir que un taller consulte datos de otro taller.
12. Ejecutar pruebas backend, frontend y de migraciones en CI.

### Fuera del Sprint 1

Los siguientes elementos se dejan para el Sprint 2 o posterior:

- Registro de avances productivos.
- Correcciones de cantidades.
- Cambio manual de etapas.
- Pausa, reanudación y cancelación.
- Evidencias y Supabase Storage.
- Tracking público para clientes.
- Dashboard operativo completo.
- Administración de equipo e invitaciones.
- Pruebas Playwright del recorrido completo.
- Despliegue en Vercel, Render y Supabase productivo.

No se adelantarán elementos fuera de alcance mientras exista trabajo comprometido sin pruebas o sin integrar.

## 3. Restricciones globales

- Flyway es la única herramienta autorizada para modificar el esquema; no usar `GORM AutoMigrate`.
- Toda consulta de negocio debe filtrar por `workshop_id`.
- Un UUID de otro taller responde `404`, no `403`.
- El frontend consume la API; no consulta PostgreSQL directamente.
- Supabase Auth es la única fuente de autenticación.
- Los DTO no deben exponer `tracking_token`, secretos o campos internos.
- Todos los endpoints usan `/api/v1`.
- Las respuestas exitosas usan `{ "data": ..., "meta": ... }`.
- Los errores usan `{ "error": { "code", "message", "details", "request_id" } }`.
- Cada entregable debe incluir pruebas y pasar los comandos de su sección antes del pull request.
- Nadie realiza push directo a `main`.

## 4. Distribución del equipo

| Persona | Rama | Responsabilidad primaria | Tareas del plan maestro |
|---|---|---|---|
| José | `feat/s1-jose-platform-auth` | Liderazgo técnico, base Go, autenticación, composición e integración | Tarea 1, backend de tarea 4 y parte inicial de tarea 16 |
| Lucero | `feat/s1-lucero-frontend` | Aplicación Vue, sistema visual, login, clientes y órdenes | Tarea 3; frontend de tareas 4–5; tarea 7 |
| Michael | `feat/s1-michael-clients-orders` | Contrato HTTP y dominio backend de clientes y órdenes | Tareas 5–6 backend |
| Stefano | `feat/s1-stefano-data-devops` | PostgreSQL, Flyway, Docker Compose y CI inicial | Tarea 2 y base de tarea 18 |

### Responsabilidad compartida

Todos son responsables de:

- Mantener su rama actualizada con `main`.
- No romper contratos publicados.
- Revisar al menos un pull request de otro integrante.
- Resolver comentarios antes de solicitar merge.
- Actualizar este documento marcando las casillas completadas.
- Avisar en la reunión diaria si una dependencia puede retrasar a otra persona.

## 5. Estrategia de ramas

### 5.1 Preparación común

Cada integrante ejecuta:

```powershell
git switch main
git pull --ff-only origin main
git status
```

`git status` debe mostrar un árbol limpio antes de crear la rama. La carpeta local `.superpowers/` no debe añadirse a ningún commit.

### 5.2 Creación de ramas

#### José

```powershell
git switch main
git pull --ff-only origin main
git switch -c feat/s1-jose-platform-auth
git push -u origin feat/s1-jose-platform-auth
```

#### Lucero

```powershell
git switch main
git pull --ff-only origin main
git switch -c feat/s1-lucero-frontend
git push -u origin feat/s1-lucero-frontend
```

#### Michael

```powershell
git switch main
git pull --ff-only origin main
git switch -c feat/s1-michael-clients-orders
git push -u origin feat/s1-michael-clients-orders
```

#### Stefano

```powershell
git switch main
git pull --ff-only origin main
git switch -c feat/s1-stefano-data-devops
git push -u origin feat/s1-stefano-data-devops
```

### 5.3 Actualización diaria de una rama personal

Cada persona debe guardar o confirmar su trabajo antes de sincronizar:

```powershell
git status
git fetch origin
git rebase origin/main
git push --force-with-lease
```

`--force-with-lease` se permite únicamente sobre la rama personal. Nunca debe usarse sobre `main` ni sobre la rama de otra persona.

Si el equipo prefiere evitar reescritura de historial, puede sustituir los dos últimos comandos por:

```powershell
git merge origin/main
git push
```

El equipo debe elegir una sola variante al iniciar el sprint y utilizarla consistentemente.

### 5.4 Reglas de pull requests

- Un PR representa un entregable revisable, no todo el sprint.
- Tamaño recomendado: menos de 500 líneas productivas, excluyendo archivos generados y `package-lock.json`.
- Título: `tipo(área): resultado`, por ejemplo `feat(auth): validate Supabase JWT`.
- El cuerpo explica objetivo, archivos relevantes, pruebas ejecutadas y dependencias.
- Adjuntar captura o video corto para cambios visuales.
- El autor no aprueba ni fusiona su propio PR.
- Se requiere al menos una aprobación.
- El CI debe estar verde.
- Usar `Create a merge commit` para conservar la rama personal entre entregables; no usar squash en estas cuatro ramas persistentes.
- Después de cada merge, todos actualizan sus ramas desde `main`.

## 6. Propiedad de archivos y prevención de conflictos

| Ruta o archivo | Propietario | Regla |
|---|---|---|
| `backend/cmd/server/` | José | Solo José realiza el cableado final de módulos |
| `backend/internal/platform/` | José | Otros proponen cambios mediante comentario o PR pequeño |
| `backend/internal/auth/` | José | Michael puede consumir tipos públicos, no modificarlos sin coordinación |
| `backend/internal/clients/` | Michael | José revisa seguridad e integración |
| `backend/internal/orders/` | Michael | José revisa transacciones y tenant isolation |
| `backend/go.mod`, `backend/go.sum` | José | Dependencias nuevas se coordinan antes de agregarlas |
| `frontend/` | Lucero | Ningún otro integrante modifica frontend durante Sprint 1 |
| `database/migrations/` | Stefano | Solo Stefano asigna números de migración |
| `compose.yaml` | Stefano | Cambios de servicios se coordinan con José y Lucero |
| `.env.example` | Stefano | Cada propietario comunica nuevas variables antes de usarlas |
| `.github/workflows/` | Stefano | José y Michael revisan comandos backend/database |
| `docs/contracts/sprint-01-api.md` | Michael | José aprueba cambios; Lucero valida que sea consumible |
| `README.md` | Stefano | Se actualiza al cierre con comandos realmente verificados |

### Regla para registrar rutas backend

Para evitar que José y Michael editen simultáneamente `router.go`, cada módulo de Michael debe exponer:

```go
func RegisterRoutes(group *gin.RouterGroup, handler *Handler)
```

Michael prueba esa función dentro de su módulo. José se encarga de invocarla desde la composición principal una vez fusionado el PR.

### Regla para migraciones

- Solo Stefano crea archivos `Vn__description.sql`.
- Michael entrega a Stefano el cambio requerido mediante una tabla con columna, tipo, nulabilidad, restricción e índice.
- Una migración ya fusionada no se edita.
- Si aparece un cambio, Stefano crea la siguiente versión.

## 7. Contrato HTTP congelado para Sprint 1

Michael documentará estos contratos en `docs/contracts/sprint-01-api.md` durante el primer día. José valida autenticación y errores; Lucero valida que los DTO sean suficientes para las pantallas.

### 7.1 Cabeceras privadas

```http
Authorization: Bearer <supabase-jwt>
Content-Type: application/json
X-Request-ID: <uuid-opcional>
```

### 7.2 Sesión

```http
GET /api/v1/me
```

```json
{
  "data": {
    "id": "0cce94d7-ef6a-43ce-9aad-f41deed2f576",
    "name": "José",
    "email": "jose@example.com",
    "workshop": {
      "id": "3b35c506-592d-4d33-94fb-66c23912ad35",
      "name": "Taller Demo",
      "role": "OWNER"
    }
  },
  "meta": {}
}
```

```http
GET /api/v1/workshops/current
```

### 7.3 Clientes

```http
GET   /api/v1/clients
POST  /api/v1/clients
GET   /api/v1/clients/{id}
PATCH /api/v1/clients/{id}
```

Solicitud de creación:

```json
{
  "name": "María Torres",
  "company_name": "Moda Sur",
  "phone": "+51 999 999 999",
  "email": "maria@example.com"
}
```

Respuesta:

```json
{
  "data": {
    "id": "6b081bbc-0d40-4800-aa58-2ae186afcbcd",
    "name": "María Torres",
    "company_name": "Moda Sur",
    "phone": "+51 999 999 999",
    "email": "maria@example.com",
    "is_active": true
  },
  "meta": {}
}
```

### 7.4 Órdenes

```http
GET   /api/v1/orders
POST  /api/v1/orders
GET   /api/v1/orders/{id}
PATCH /api/v1/orders/{id}
```

Solicitud de creación:

```json
{
  "client_id": "6b081bbc-0d40-4800-aa58-2ae186afcbcd",
  "product": "Polos de algodón",
  "description": "Color azul marino, talla surtida",
  "quantity": 500,
  "start_date": "2026-09-21",
  "delivery_date": "2026-10-05",
  "responsible_user_id": null,
  "applicable_stages": [
    "CORTE",
    "CONFECCION",
    "ACABADO",
    "CONTROL_CALIDAD",
    "ENTREGA"
  ]
}
```

Respuesta resumida:

```json
{
  "data": {
    "id": "dc987722-51b0-4719-af8e-40e8609b6f64",
    "code": "TF-0001",
    "client": {
      "id": "6b081bbc-0d40-4800-aa58-2ae186afcbcd",
      "name": "María Torres"
    },
    "product": "Polos de algodón",
    "quantity": 500,
    "status": "ACTIVE",
    "current_stage": {
      "code": "CORTE",
      "name": "Corte",
      "status": "IN_PROGRESS"
    },
    "delivery_date": "2026-10-05",
    "progress_percent": 16.67
  },
  "meta": {}
}
```

Filtros admitidos:

```text
status
client_id
stage
delivery_from
delivery_to
page
page_size
```

### 7.5 Error común

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Revisa los campos enviados.",
    "details": {
      "delivery_date": "Debe ser igual o posterior a la fecha de inicio."
    },
    "request_id": "81ade11b-e729-4dbc-b1e0-f3ae013bf4ed"
  }
}
```

### 7.6 Estados HTTP del sprint

| Código | Uso |
|---:|---|
| 200 | Consulta o actualización correcta |
| 201 | Cliente u orden creada |
| 400 | JSON, filtro o campo inválido |
| 401 | JWT ausente, expirado o inválido |
| 403 | Rol sin permiso |
| 404 | Recurso inexistente o de otro taller |
| 409 | Conflicto de código, estado o concurrencia |
| 422 | Regla de negocio incumplida |
| 500 | Error inesperado con `request_id` |

Los contratos quedan congelados al finalizar el segundo día. Después de ese punto, cualquier cambio requiere aprobación de José, Michael y Lucero y debe actualizar primero `docs/contracts/sprint-01-api.md`.

## 8. Trabajo de José — Plataforma y autenticación

**Rama:** `feat/s1-jose-platform-auth`

**Objetivo personal:** entregar una API Go arrancable, segura y preparada para conectar los módulos de Michael.

### Archivos principales

```text
backend/go.mod
backend/go.sum
backend/cmd/server/main.go
backend/cmd/bootstrap/main.go
backend/internal/platform/config/
backend/internal/platform/database/
backend/internal/platform/httpx/
backend/internal/auth/
backend/.env.example
backend/Dockerfile
```

### J1. Base Go y health checks

- [ ] Inicializar `github.com/Josed20/Tallerflow/backend`.
- [ ] Agregar Gin, GORM PostgreSQL, UUID y Testify.
- [ ] Implementar `config.Load()` con validación de variables obligatorias.
- [ ] Implementar `database.Open(databaseURL)` sin AutoMigrate.
- [ ] Implementar `GET /health/live`.
- [ ] Implementar `GET /health/ready` con `SELECT 1`.
- [ ] Configurar `http.Server` con timeouts de lectura, escritura e inactividad.
- [ ] Crear pruebas unitarias de configuración y health checks.
- [ ] Abrir PR `feat(platform): bootstrap Go API and health checks`.

Comprobación:

```powershell
cd backend
go test ./internal/platform/...
go vet ./...
```

### J2. Autenticación y principal

- [ ] Definir roles `OWNER`, `ADMIN` y `OPERATOR`.
- [ ] Definir `auth.Identity` y `auth.Principal`.
- [ ] Validar firma, `iss`, `aud` y `exp` del JWT Supabase.
- [ ] Resolver usuario y membresía activa desde PostgreSQL.
- [ ] Responder `401` para token inválido y `403` para membresía/rol insuficiente.
- [ ] Implementar `auth.Require` y `auth.RequireRoles`.
- [ ] Implementar `GET /api/v1/me`.
- [ ] Implementar `GET /api/v1/workshops/current`.
- [ ] Implementar bootstrap no público del primer OWNER.
- [ ] Probar token expirado, membresía inactiva y rol incorrecto.
- [ ] Abrir PR `feat(auth): add Supabase authentication and tenant context`.

Comprobación:

```powershell
cd backend
go test ./internal/auth/... ./internal/platform/httpx/...
go vet ./...
```

### J3. Contrato HTTP e integración de módulos

- [ ] Implementar `request_id` por petición.
- [ ] Implementar el contrato estándar de errores.
- [ ] Crear grupo privado `/api/v1` protegido por JWT.
- [ ] Registrar `clients.RegisterRoutes` después del merge de Michael.
- [ ] Registrar `orders.RegisterRoutes` después del merge de Michael.
- [ ] Confirmar que recursos ajenos al taller se traduzcan a `404`.
- [ ] Ejecutar pruebas de todo el backend tras cada integración.

Comprobación:

```powershell
cd backend
go test ./... -race
go vet ./...
```

### José no debe modificar

- `database/migrations/` sin coordinación con Stefano.
- Componentes o vistas de `frontend/`.
- Reglas internas de clientes u órdenes sin revisión de Michael.

## 9. Trabajo de Lucero — Frontend completo del sprint

**Rama:** `feat/s1-lucero-frontend`

**Objetivo personal:** entregar una interfaz responsive alineada con Stitch para login, clientes, creación y listado de órdenes.

### Archivos principales

```text
frontend/package.json
frontend/vite.config.js
frontend/vitest.config.js
frontend/tailwind.config.js
frontend/src/main.js
frontend/src/App.vue
frontend/src/assets/main.css
frontend/src/router/
frontend/src/layouts/
frontend/src/components/
frontend/src/services/
frontend/src/modules/auth/
frontend/src/modules/clients/
frontend/src/modules/orders/
frontend/.env.example
frontend/Dockerfile
```

### L1. Shell Vue y componentes base

- [ ] Crear Vue 3 con Vite y JavaScript.
- [ ] Configurar Vue Router, Pinia, Axios y Tailwind.
- [ ] Configurar Vitest, Vue Test Utils y entorno DOM.
- [ ] Configurar ESLint y el script `npm run lint`.
- [ ] Implementar `AppLayout` con navegación responsive.
- [ ] Crear `BaseButton`, `BaseField` y `ViewState`.
- [ ] Implementar estilos tomando azul marino, turquesa y fondos claros de Stitch.
- [ ] Garantizar controles táctiles de al menos 44 px.
- [ ] Implementar estados cargando, vacío, sin resultados y error.
- [ ] Abrir PR `feat(frontend): bootstrap Vue application shell`.

Comprobación:

```powershell
cd frontend
npm run test -- --run
npm run build
```

### L2. Login y sesión

- [ ] Configurar el cliente oficial de Supabase.
- [ ] Implementar store Pinia con `signIn`, `restore`, `loadMe` y `signOut`.
- [ ] Inyectar el access token en Axios.
- [ ] Implementar `LoginView` según Stitch.
- [ ] Proteger rutas con `meta.requiresAuth`.
- [ ] Redirigir a `/login` cuando no exista sesión.
- [ ] Mostrar sesión vencida y credenciales inválidas con mensajes distintos.
- [ ] Probar navegación exitosa y error de autenticación.
- [ ] Abrir PR `feat(frontend-auth): add login and protected routes`.

### L3. Clientes

- [ ] Implementar servicio `clients.list/create/get/update`.
- [ ] Implementar `ClientsView`.
- [ ] Implementar `ClientForm` reutilizable.
- [ ] Validar nombre obligatorio y formato de email.
- [ ] Permitir crear cliente dentro del flujo de orden.
- [ ] Mostrar cliente inactivo sin borrarlo físicamente.
- [ ] Probar formulario, estados vacíos y error de API.
- [ ] Abrir PR `feat(frontend-clients): add client management`.

### L4. Órdenes

- [ ] Implementar servicio `orders.list/create/get/update`.
- [ ] Implementar `OrdersView` con filtros y paginación.
- [ ] Mostrar tabla en desktop y tarjetas en móvil.
- [ ] Implementar `CreateOrderView` y `OrderForm`.
- [ ] Seleccionar o crear cliente.
- [ ] Capturar producto, descripción, cantidad y fechas.
- [ ] Permitir elegir etapas aplicables manteniendo `ENTREGA` obligatoria.
- [ ] Validar cantidad positiva y entrega no anterior al inicio.
- [ ] Navegar al detalle después de crear la orden.
- [ ] Probar payload exacto y sincronización de filtros con la URL.
- [ ] Abrir PR `feat(frontend-orders): add order list and creation`.

### L5. Integración visual final

- [ ] Conectar las vistas con la API integrada.
- [ ] Eliminar fixtures temporales usados durante desarrollo.
- [ ] Verificar 360 px, 768 px y 1440 px de ancho.
- [ ] Verificar navegación completa solo con teclado.
- [ ] Mostrar correctamente errores `401`, `403`, `404`, `409` y `422`.
- [ ] Adjuntar capturas desktop y móvil al PR final.

Comprobación:

```powershell
cd frontend
npm run lint
npm run test -- --run
npm run build
```

### Lucero no debe modificar

- Migraciones SQL.
- Modelos GORM.
- Formato del contrato sin actualizar primero el documento compartido.

## 10. Trabajo de Michael — Clientes y órdenes backend

**Rama:** `feat/s1-michael-clients-orders`

**Objetivo personal:** entregar los módulos de negocio que permiten crear clientes y órdenes respetando aislamiento, roles y transacciones.

### Archivos principales

```text
docs/contracts/sprint-01-api.md
backend/internal/clients/
backend/internal/orders/
```

Cada módulo debe contener:

```text
model.go
dto.go
repository.go
service.go
handler.go
routes.go
service_test.go
handler_test.go
repository_integration_test.go
```

### M1. Contrato API

- [ ] Crear `docs/contracts/sprint-01-api.md` con los ejemplos de la sección 7.
- [ ] Confirmar nombres de campos con Lucero.
- [ ] Confirmar errores, roles y middleware con José.
- [ ] Congelar el contrato al finalizar el segundo día.
- [ ] Abrir PR `docs(api): define Sprint 1 HTTP contract`.

### M2. Clientes backend

- [ ] Definir `Client`, `CreateInput`, `UpdateInput` y DTO de salida.
- [ ] Implementar repositorio filtrado siempre por `workshop_id`.
- [ ] Implementar `List`, `Create`, `Get` y `Update`.
- [ ] Impedir eliminación física; usar `is_active=false`.
- [ ] Permitir escritura solo a OWNER/ADMIN.
- [ ] Devolver `404` cuando el UUID pertenezca a otro taller.
- [ ] Exponer `clients.RegisterRoutes` sin editar el router central.
- [ ] Probar nombre vacío, email inválido, operador sin permiso y otro taller.
- [ ] Abrir PR `feat(clients): manage workshop clients`.

Comprobación:

```powershell
cd backend
go test ./internal/clients/... -race
```

### M3. Órdenes backend

- [ ] Definir `Order`, `Stage`, `CreateInput`, `UpdateInput` y `ListFilter`.
- [ ] Validar cliente del mismo taller.
- [ ] Validar cantidad positiva y fechas coherentes.
- [ ] Bloquear `workshop_counters` con `FOR UPDATE`.
- [ ] Generar `TF-0001`, `TF-0002` y siguientes por taller.
- [ ] Generar token de tracking de 32 bytes aunque no se exponga todavía.
- [ ] Crear las siete etapas en la misma transacción.
- [ ] Crear `PEDIDO_CONFIRMADO` como `COMPLETED`.
- [ ] Crear la primera etapa aplicable como `IN_PROGRESS`.
- [ ] Marcar etapas no aplicables como `SKIPPED`.
- [ ] Implementar listado con filtros y máximo de 100 registros por página.
- [ ] Implementar edición limitada a producto, descripción, entrega y responsable.
- [ ] Exponer `orders.RegisterRoutes` sin editar el router central.
- [ ] Abrir PR `feat(orders): create and list textile orders`.

Pruebas obligatorias:

```text
Creación estándar con siete etapas
Cliente de otro taller => 404
Código concurrente sin duplicados
OPERATOR creando orden => 403
Entrega anterior al inicio => 422
Orden completada no editable => 409
Page size mayor de 100 => normalizado o rechazado consistentemente
```

Comprobación:

```powershell
cd backend
go test ./internal/orders/... -race
```

### M4. Revisión cruzada

- [ ] Revisar el PR de autenticación de José.
- [ ] Comprobar que `auth.Principal` se use sin duplicar tipos.
- [ ] Entregar a Stefano cualquier cambio de esquema en formato explícito.
- [ ] Ayudar a Lucero con ejemplos reales de respuestas.

### Michael no debe modificar

- `backend/internal/platform/httpx/router.go`.
- Números o contenido de migraciones ya creadas.
- Componentes Vue.

## 11. Trabajo de Stefano — Base de datos, Docker y CI

**Rama:** `feat/s1-stefano-data-devops`

**Objetivo personal:** garantizar que cualquier integrante pueda clonar el repositorio, levantarlo y obtener el mismo esquema y resultados de pruebas.

### Archivos principales

```text
database/migrations/
backend/internal/platform/database/schema_integration_test.go
compose.yaml
.env.example
.github/workflows/ci.yml
README.md
```

### S1. Migraciones Flyway

- [ ] Crear `V1__create_extensions.sql` con `citext`.
- [ ] Crear `V2__create_identity_and_workshops.sql`.
- [ ] Crear `V3__create_clients.sql`.
- [ ] Crear `V4__create_orders_and_stages.sql`.
- [ ] Crear `V5__create_updates_and_attachments.sql` para dejar el esquema preparado, aunque updates/attachments no se usen aún.
- [ ] Crear `V6__add_constraints_and_indexes.sql`.
- [ ] Agregar claves foráneas, checks, índices tenant-aware e índices únicos parciales.
- [ ] Probar migración sobre base vacía.
- [ ] Probar que una segunda ejecución no realice cambios.
- [ ] Abrir PR `feat(database): define TallerFlow schema with Flyway`.

Comprobación:

```powershell
docker compose up -d postgres
docker compose run --rm flyway migrate
docker compose run --rm flyway validate
docker compose run --rm flyway migrate
```

Resultado esperado de la última orden:

```text
Schema is up to date. No migration necessary.
```

### S2. Docker Compose

- [ ] Configurar `postgres` con volumen y health check.
- [ ] Configurar `flyway` esperando a PostgreSQL saludable.
- [ ] Configurar `backend` esperando a Flyway completado correctamente.
- [ ] Configurar `frontend` esperando al backend para el entorno local.
- [ ] Exponer únicamente los puertos necesarios.
- [ ] Agregar `.env.example` sin secretos reales.
- [ ] Verificar `docker compose config`.
- [ ] Verificar reconstrucción completa desde cero.
- [ ] Abrir PR `chore(docker): add reproducible local environment`.

Comprobación:

```powershell
docker compose config
docker compose up --build -d
docker compose ps
Invoke-WebRequest http://localhost:8080/health/ready
```

### S3. Prueba automática del esquema

- [ ] Crear prueba que verifique las diez tablas principales.
- [ ] Verificar `ux_order_updates_request`.
- [ ] Verificar `ux_order_stages_current`.
- [ ] Verificar FK de `orders.current_stage_id`.
- [ ] Verificar restricciones de roles, estados y cantidades.
- [ ] Ejecutar contra PostgreSQL real, no SQLite.

### S4. CI inicial

- [ ] Crear job backend con PostgreSQL 17.
- [ ] Ejecutar Flyway validate y migrate.
- [ ] Ejecutar `go test ./... -race` y `go vet ./...`.
- [ ] Crear job frontend con Node 22.
- [ ] Ejecutar `npm ci`, lint, pruebas y build.
- [ ] Evitar secretos productivos en CI.
- [ ] Abrir PR `ci: validate database backend and frontend`.

### S5. Documentación local

- [ ] Documentar requisitos previos.
- [ ] Documentar variables de entorno.
- [ ] Documentar arranque con Docker.
- [ ] Documentar bootstrap del primer OWNER.
- [ ] Documentar comandos de pruebas.
- [ ] Verificar todos los comandos en un clon o worktree limpio.

### Stefano no debe modificar

- Reglas de negocio dentro de servicios Go.
- DTO publicados sin aprobación de Michael.
- Componentes frontend.

## 12. Oleadas de ejecución

### Oleada 1 — Días 1 y 2: contratos y cimientos

| Persona | Entregable |
|---|---|
| José | Go module, configuración, conexión y health checks |
| Lucero | Vue/Vite, router, Pinia, Tailwind, componentes base |
| Michael | Contrato HTTP congelado y diseño de DTO |
| Stefano | PostgreSQL, Flyway V1–V6 y primer Compose |

Orden recomendado de merge:

1. Michael: contrato HTTP.
2. José: backend base.
3. Stefano: migraciones y PostgreSQL.
4. Lucero: frontend base.

Punto de control:

```text
Backend compila
Frontend compila
PostgreSQL está saludable
Flyway migra una base vacía
Contrato HTTP aprobado por José, Lucero y Michael
```

### Oleada 2 — Días 3 a 5: autenticación y dominio

| Persona | Entregable |
|---|---|
| José | JWT, principal, roles, `/me`, taller actual y bootstrap |
| Lucero | Login, store de sesión y guardas del router |
| Michael | Clientes backend y pruebas tenant-aware |
| Stefano | Compose integrado, prueba del esquema y soporte de migraciones |

Orden recomendado de merge:

1. Stefano: esquema final necesario para auth/clientes.
2. José: autenticación backend.
3. Michael: clientes backend.
4. Lucero: login y sesión.

Punto de control:

```text
OWNER inicia sesión
GET /api/v1/me responde con taller y rol
OWNER crea y lista clientes
OPERATOR no puede crear clientes
Otro taller recibe 404
```

### Oleada 3 — Días 6 a 8: primera orden

| Persona | Entregable |
|---|---|
| José | Registro de rutas y normalización de errores |
| Lucero | Clientes UI, lista de órdenes y formulario de creación |
| Michael | Crear/listar/editar órdenes y etapas transaccionales |
| Stefano | CI backend/frontend/database y documentación local |

Orden recomendado de merge:

1. Michael: órdenes backend.
2. José: cableado y errores.
3. Lucero: clientes y órdenes frontend.
4. Stefano: CI sobre el sistema integrado.

Punto de control:

```text
OWNER crea una orden
Código secuencial generado sin colisiones
Se crean siete etapas
Listado desktop y móvil muestran la orden
Filtros actualizan la URL
CI está verde
```

### Oleada 4 — Días 9 y 10: estabilización y demo

Todo el equipo trabaja sobre defectos encontrados, manteniendo la propiedad de archivos.

- [ ] José ejecuta y revisa todas las pruebas backend.
- [ ] Lucero verifica responsive, teclado y estados de interfaz.
- [ ] Michael revisa consultas tenant-aware y transacciones.
- [ ] Stefano levanta el sistema desde una base y volúmenes vacíos.
- [ ] El equipo ejecuta el recorrido de demostración dos veces.
- [ ] Se corrigen defectos críticos y altos.
- [ ] Se actualiza README con comandos confirmados.
- [ ] Se etiqueta el cierre como `sprint-1` después de la aprobación del equipo.

## 13. Matriz de dependencias

| Entregable | Depende de | Bloquea a |
|---|---|---|
| Contrato HTTP | Arquitectura aprobada | Lucero, José y Michael |
| Backend base | Ninguno | Auth, clientes, órdenes, Docker backend |
| Migraciones | Especificación de datos | Auth, clientes y órdenes integración |
| Frontend base | Contrato visual | Login, clientes y órdenes UI |
| Auth backend | Backend base + migraciones | Login integrado y endpoints privados |
| Clientes backend | Auth types + migración clients | Clientes UI y creación de orden |
| Órdenes backend | Clientes + migraciones orders/stages | Orden UI |
| Docker completo | Backend/frontend Dockerfiles | Demo local |
| CI | Comandos estables de las tres capas | Merge final |

Una dependencia bloqueada no autoriza a trabajar fuera del sprint. La persona debe adelantar pruebas, documentación, revisión o fixtures del mismo entregable.

## 14. Revisión de pull requests

| Autor | Revisor principal | Foco |
|---|---|---|
| José | Michael | Interfaces, autenticación, errores y testabilidad |
| Lucero | José | Contrato API, permisos, estados y accesibilidad básica |
| Michael | José | Multi-tenancy, transacciones, roles y concurrencia |
| Stefano | Michael | Restricciones SQL, migraciones y comandos reproducibles |

Lucero debe revisar el contrato de cualquier PR backend que cambie un DTO consumido por frontend. Stefano debe revisar cualquier PR que requiera una columna, índice o variable de entorno nueva.

### Checklist obligatorio del autor

```markdown
- [ ] La rama parte de `main` actualizado.
- [ ] El cambio pertenece al alcance de Sprint 1.
- [ ] Incluye pruebas nuevas o explica por qué no corresponden.
- [ ] Ejecuté las pruebas indicadas localmente.
- [ ] No agregué secretos ni archivos `.env` reales.
- [ ] No usé `AutoMigrate`.
- [ ] No expuse datos de otro taller.
- [ ] Actualicé contrato o documentación cuando cambió una interfaz.
- [ ] El CI está verde.
```

### Checklist obligatorio del revisor

```markdown
- [ ] El cambio cumple el contrato del sprint.
- [ ] Los nombres y tipos coinciden entre capas.
- [ ] Las consultas incluyen `workshop_id`.
- [ ] Los errores no filtran existencia de otro taller.
- [ ] Las pruebas fallarían si la regla principal se rompe.
- [ ] No se incorporó alcance del Sprint 2.
- [ ] El código puede integrarse sin editar archivos de otro propietario.
```

## 15. Convención de commits

Formato:

```text
tipo(área): resultado concreto
```

Ejemplos:

```text
feat(platform): add API health checks
feat(auth): validate Supabase JWT
feat(clients): create tenant-scoped client API
feat(orders): create standard production stages
feat(frontend-auth): add login flow
feat(frontend-orders): add responsive order list
feat(database): add initial Flyway migrations
ci: run database backend and frontend checks
test(orders): cover concurrent order numbering
docs(api): freeze Sprint 1 contracts
```

No usar mensajes como `cambios`, `avance`, `fix`, `update` o `prueba` sin explicar el resultado.

## 16. Coordinación diaria

Reunión diaria máxima de 15 minutos. Cada persona responde:

1. Qué entregable terminó desde la última reunión.
2. Qué entregable terminará hoy.
3. Qué dependencia o decisión la bloquea.
4. Qué PR necesita revisión.

Después de la reunión:

- Los bloqueos de contrato los resuelven José, Michael y Lucero el mismo día.
- Los bloqueos de migración los resuelven Stefano y Michael.
- Los PR listos se revisan antes de iniciar trabajo nuevo.
- Nadie mantiene un PR listo sin revisión durante más de un día hábil.

## 17. Riesgos y respuestas

| Riesgo | Señal temprana | Respuesta |
|---|---|---|
| Contrato frontend/backend cambia repetidamente | Campos renombrados después del día 2 | Congelar contrato y exigir aprobación de tres responsables |
| Conflictos en router Go | Michael y José editan el mismo archivo | Michael exporta `RegisterRoutes`; José hace el cableado |
| Conflictos de migración | Dos archivos usan el mismo número V | Stefano es propietario único de numeración |
| Lucero queda bloqueada por API | Endpoint aún no fusionado | Usar Axios mock con el JSON exacto del contrato |
| Michael queda bloqueado por auth | Middleware aún no fusionado | Probar servicios con `auth.Principal` construido en tests |
| Docker funciona solo en una máquina | Variables implícitas o rutas locales | `.env.example`, health checks y prueba desde clon limpio |
| Sprint acumula alcance | Se inicia tracking o evidencias | Detener trabajo y devolverlo al backlog de Sprint 2 |
| PR demasiado grande | Más de un módulo o varios objetivos | Dividir por entregable antes de revisión |

## 18. Pruebas mínimas del Sprint 1

### Backend

```powershell
cd backend
go test ./... -race
go vet ./...
```

Debe cubrir:

- Configuración inválida.
- Health ready con DB disponible y no disponible.
- JWT expirado o inválido.
- Membresía inactiva.
- Permisos OWNER/ADMIN/OPERATOR.
- Cliente de otro taller.
- Código de orden concurrente.
- Etapas aplicables y omitidas.
- Validaciones de fechas y cantidades.

### Frontend

```powershell
cd frontend
npm run lint
npm run test -- --run
npm run build
```

Debe cubrir:

- Login correcto e incorrecto.
- Guarda de rutas.
- Token enviado por Axios.
- Formulario de cliente.
- Formulario de orden.
- Fechas inválidas.
- Etapas seleccionadas.
- Filtros del listado.
- Estados cargando, vacío y error.

### Base de datos

```powershell
docker compose up -d postgres
docker compose run --rm flyway validate
docker compose run --rm flyway migrate
```

Debe comprobar:

- Diez tablas.
- Claves foráneas.
- Checks de estados y roles.
- Código único por taller.
- Una sola etapa actual por orden.
- Idempotencia preparada para avances futuros.

### Integración local

```powershell
docker compose down -v
docker compose up --build -d
docker compose ps
Invoke-WebRequest http://localhost:8080/health/ready
```

## 19. Definición de terminado individual

Una tarea se considera terminada cuando:

- El código está en la rama personal correcta.
- Las pruebas del módulo pasan.
- El formatter y linter pasan.
- El contrato está actualizado.
- No contiene secretos.
- Tiene un commit descriptivo.
- El PR tiene aprobación.
- El CI está verde.
- El cambio está fusionado en `main`.
- El autor sincronizó nuevamente su rama con `main`.

Código únicamente presente en una rama personal no se considera terminado.

## 20. Definición de terminado del Sprint 1

El sprint se acepta únicamente si, desde un entorno limpio:

1. `docker compose up --build` levanta todos los servicios.
2. Flyway aplica V1–V6 sin errores.
3. `/health/live` y `/health/ready` responden correctamente.
4. Un OWNER inicia sesión.
5. El OWNER crea un cliente.
6. El OWNER crea una orden de 500 unidades.
7. La orden obtiene un código `TF-xxxx`.
8. Se crean siete etapas con estados correctos.
9. La orden aparece en el listado desktop y móvil.
10. Los filtros del listado funcionan.
11. Un OPERATOR no puede crear clientes ni órdenes.
12. Otro taller no puede consultar esos recursos.
13. Pruebas backend, frontend y base de datos pasan en CI.
14. README contiene instrucciones verificadas.
15. No existe `AutoMigrate` en el repositorio.

## 21. Guion de demostración

La demo final debe durar menos de diez minutos:

1. Stefano muestra el arranque con Docker y Flyway.
2. José muestra health checks, login y usuario/taller actual.
3. Lucero crea un cliente desde la interfaz.
4. Lucero crea una orden de 500 polos.
5. Michael explica el código secuencial y las siete etapas en PostgreSQL/API.
6. Lucero muestra listado, filtros y versión móvil.
7. José demuestra que OPERATOR recibe `403` y otro taller recibe `404`.
8. Stefano muestra el pipeline CI en verde.

## 22. Cierre y siguiente sprint

Después de aprobar la demo:

- Fusionar los últimos PR aprobados.
- Crear la etiqueta Git `sprint-1`.
- Registrar defectos no críticos en el backlog.
- No transportar trabajo parcialmente integrado como si estuviera terminado.
- Preparar Sprint 2 con detalle de orden, avances incrementales, correcciones, etapas, pausa/reanudación y evidencias.

El Sprint 2 debe comenzar desde el commit etiquetado de Sprint 1 y con las cuatro ramas personales sincronizadas nuevamente desde `main`.
