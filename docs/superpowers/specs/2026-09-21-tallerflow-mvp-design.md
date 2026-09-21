# TallerFlow MVP — Especificación de diseño

**Estado:** Aprobado para planificación
**Fecha:** 2026-09-21
**Producto:** TallerFlow
**Alcance:** MVP de trazabilidad de órdenes para talleres textiles

## 1. Propósito

TallerFlow permite que un taller de confección registre y consulte el recorrido de una orden desde su confirmación hasta la entrega. El MVP debe demostrar que un taller puede controlar mejor sus pedidos cuando los operarios registran avances y el dueño, el administrador y el cliente autorizado obtienen visibilidad sin depender de llamadas, mensajes o cuadernos.

La experiencia se resume así:

- El operario registra.
- El dueño o administrador controla.
- El cliente consulta mediante un enlace privado.

El producto está dirigido inicialmente a talleres pequeños y medianos de Gamarra y Lima, especialmente aquellos con 10 a 30 operarios y varias órdenes simultáneas.

## 2. Criterios de éxito del MVP

El primer corte debe permitir completar de extremo a extremo este flujo:

1. Iniciar sesión.
2. Crear o seleccionar un cliente.
3. Crear una orden con sus etapas aplicables.
4. Asignar un responsable.
5. Registrar avances incrementales.
6. Consultar el historial.
7. Publicar una evidencia autorizada.
8. Abrir el tracking externo mediante un token privado.
9. Pausar, reanudar, cancelar o completar la orden según los permisos.

La creación de una orden debe tomar menos de cinco minutos y el registro de un avance debe poder realizarse en segundos desde un teléfono.

## 3. Fuera de alcance

El MVP no incluye:

- Contabilidad, facturación electrónica, planillas o POS.
- Compras, inventario avanzado o gestión de proveedores.
- CRM completo, marketplace, logística o delivery.
- Cobros, suscripciones o planes comerciales.
- Aplicaciones móviles nativas.
- Integración técnica con Gamarra Mayoristas.
- Analítica industrial, exportaciones o reportes avanzados.
- Trabajo offline o sincronización local.
- Predicción de retrasos, inteligencia artificial o machine learning.
- WebSockets, Redis, Kafka, Kubernetes, microservicios, CQRS o event sourcing.

## 4. Fuentes visuales de referencia

Los diseños de Stitch se utilizan como referencia visual, no como código ni como contrato técnico. Para la implementación deben tomarse únicamente las variantes vigentes de:

- Login.
- Dashboard del dueño.
- Listado de órdenes desktop y móvil.
- Crear orden.
- Detalle de orden desktop y móvil.
- Detalle de orden pausada.
- Registrar avance del operario.
- Clientes.
- Equipo y usuarios.
- Tracking externo corregido.
- Estados cargando, vacío, sin resultados y error.
- Confirmar corrección y confirmar cancelación.

Los frames antiguos o duplicados de Stitch se consideran borradores y se ignoran.

## 5. Arquitectura general

La solución será un monolito modular con frontend separado:

```text
Usuario interno                    Cliente externo
      |                                   |
      v                                   v
Vue 3 + Vite                    /tracking/{token}
      |                                   |
      +----------- HTTPS / REST ----------+
                          |
                          v
                  Go + Gin + GORM
                  monolito modular
                          |
           +--------------+---------------+
           |              |               |
           v              v               v
      PostgreSQL     Supabase Auth   Supabase Storage
       Flyway          usuarios        evidencias
```

Decisiones principales:

- Frontend: Vue 3, Vite, JavaScript, Vue Router, Pinia, Axios y Tailwind CSS.
- Backend: Go, Gin, GORM y API REST bajo `/api/v1`.
- Base de datos: PostgreSQL.
- Autenticación: Supabase Auth.
- Evidencias: Supabase Storage, mediado por la API Go.
- Esquema: migraciones SQL versionadas con Flyway.
- Desarrollo local: Docker Compose.
- Producción prevista: Vercel para frontend, Render para API y Supabase para datos, autenticación y archivos.

No se utilizará `GORM AutoMigrate`. Flyway es la única herramienta autorizada para modificar el esquema.

## 6. Organización del repositorio

```text
Tallerflow/
├── backend/
│   ├── cmd/
│   │   ├── server/
│   │   └── bootstrap/
│   ├── internal/
│   │   ├── auth/
│   │   ├── workshops/
│   │   ├── team/
│   │   ├── clients/
│   │   ├── orders/
│   │   ├── production/
│   │   ├── attachments/
│   │   ├── dashboard/
│   │   ├── tracking/
│   │   └── platform/
│   └── go.mod
├── frontend/
│   └── src/
│       ├── modules/
│       ├── components/
│       ├── layouts/
│       ├── router/
│       ├── stores/
│       ├── services/
│       └── utils/
├── database/
│   └── migrations/
├── infra/
│   └── docker/
├── docs/
└── compose.yaml
```

Cada módulo backend contiene sus handlers, servicios, repositorios, modelos de dominio y DTO. `platform` contiene infraestructura compartida, pero ninguna regla de negocio.

En frontend, Pinia se reserva para sesión, taller actual y estado realmente compartido. Los formularios mantienen estado local.

## 7. Roles y permisos

### Roles internos

- `OWNER`: control total del taller.
- `ADMIN`: administra operación, clientes, órdenes y equipo.
- `OPERATOR`: consulta órdenes asignadas y registra avances.

La interfaz del MVP asume un taller activo por usuario. La tabla de membresías permite extender esta relación en el futuro.

| Acción | OWNER | ADMIN | OPERATOR |
|---|---:|---:|---:|
| Administrar usuarios | Sí | Sí | No |
| Crear y editar clientes | Sí | Sí | No |
| Crear y editar órdenes | Sí | Sí | No |
| Asignar responsables | Sí | Sí | No |
| Registrar avances | Sí | Sí | Solo en órdenes asignadas |
| Corregir avances | Sí | Sí | No |
| Cambiar de etapa | Sí | Sí | No |
| Pausar o reanudar | Sí | Sí | Solo reporta un problema |
| Cancelar una orden | Sí | Sí | No |
| Publicar evidencias | Sí | Sí | No |

El visitante externo no es un usuario interno. Solo accede a la proyección pública de una orden mediante un token privado.

## 8. Modelo de dominio

### 8.1 Identidad y taller

#### `users`

- `id UUID PK`
- `auth_user_id UUID UNIQUE NOT NULL`
- `name VARCHAR NOT NULL`
- `email CITEXT UNIQUE NOT NULL`
- `created_at TIMESTAMPTZ NOT NULL`
- `updated_at TIMESTAMPTZ NOT NULL`

#### `workshops`

- `id UUID PK`
- `name VARCHAR NOT NULL`
- `address VARCHAR NULL`
- `phone VARCHAR NULL`
- `created_at TIMESTAMPTZ NOT NULL`
- `updated_at TIMESTAMPTZ NOT NULL`

#### `workshop_users`

- `id UUID PK`
- `workshop_id UUID FK NOT NULL`
- `user_id UUID FK NOT NULL`
- `role VARCHAR NOT NULL`
- `status VARCHAR NOT NULL`
- `created_at TIMESTAMPTZ NOT NULL`
- `updated_at TIMESTAMPTZ NOT NULL`
- `UNIQUE (workshop_id, user_id)`

#### `user_invitations`

- `id UUID PK`
- `workshop_id UUID FK NOT NULL`
- `email CITEXT NOT NULL`
- `role VARCHAR NOT NULL`
- `status VARCHAR NOT NULL`
- `invited_by UUID FK NOT NULL`
- `expires_at TIMESTAMPTZ NOT NULL`
- `accepted_at TIMESTAMPTZ NULL`
- `created_at TIMESTAMPTZ NOT NULL`

La invitación se crea mediante la API administrativa de Supabase. En el primer acceso autenticado, la API local asocia la identidad, crea la membresía y marca la invitación como aceptada dentro de una transacción.

El primer `OWNER` se provisiona mediante `cmd/bootstrap`; no existirá un endpoint público de bootstrap.

### 8.2 Clientes

#### `clients`

- `id UUID PK`
- `workshop_id UUID FK NOT NULL`
- `name VARCHAR NOT NULL`
- `company_name VARCHAR NULL`
- `phone VARCHAR NULL`
- `email CITEXT NULL`
- `is_active BOOLEAN NOT NULL DEFAULT TRUE`
- `created_at TIMESTAMPTZ NOT NULL`
- `updated_at TIMESTAMPTZ NOT NULL`

Los clientes no se eliminan físicamente durante el MVP.

### 8.3 Órdenes

#### `workshop_counters`

- `workshop_id UUID PK/FK`
- `next_order_number BIGINT NOT NULL`

Permite generar códigos secuenciales por taller de forma transaccional, por ejemplo `TF-0001`.

#### `orders`

- `id UUID PK`
- `workshop_id UUID FK NOT NULL`
- `client_id UUID FK NOT NULL`
- `code VARCHAR NOT NULL`
- `product VARCHAR NOT NULL`
- `description TEXT NULL`
- `quantity INTEGER NOT NULL CHECK (quantity > 0)`
- `start_date DATE NOT NULL`
- `delivery_date DATE NOT NULL`
- `status VARCHAR NOT NULL`
- `current_stage_id UUID NULL`
- `responsible_user_id UUID NULL`
- `tracking_token VARCHAR UNIQUE NOT NULL`
- `tracking_enabled BOOLEAN NOT NULL DEFAULT TRUE`
- `created_by UUID FK NOT NULL`
- `created_at TIMESTAMPTZ NOT NULL`
- `updated_at TIMESTAMPTZ NOT NULL`
- `UNIQUE (workshop_id, code)`
- `CHECK (delivery_date >= start_date)`

Estados de la orden:

- `ACTIVE`
- `PAUSED`
- `COMPLETED`
- `CANCELLED`

El token de tracking es un valor aleatorio de al menos 256 bits, codificado para URL. No contiene el código de orden ni datos del cliente y nunca se escribe en logs.

### 8.4 Etapas

#### `order_stages`

- `id UUID PK`
- `order_id UUID FK NOT NULL`
- `name VARCHAR NOT NULL`
- `sequence SMALLINT NOT NULL`
- `status VARCHAR NOT NULL`
- `tracks_quantity BOOLEAN NOT NULL`
- `target_quantity INTEGER NULL`
- `assignee_user_id UUID NULL`
- `started_at TIMESTAMPTZ NULL`
- `completed_at TIMESTAMPTZ NULL`
- `created_at TIMESTAMPTZ NOT NULL`
- `updated_at TIMESTAMPTZ NOT NULL`
- `UNIQUE (order_id, sequence)`

Estados de etapa:

- `PENDING`
- `IN_PROGRESS`
- `COMPLETED`
- `BLOCKED`
- `SKIPPED`

Plantilla inicial:

1. Pedido confirmado.
2. Corte.
3. Confección.
4. Estampado o bordado.
5. Acabado.
6. Control de calidad.
7. Entrega.

Todas las etapas se crean con la orden. Las no aplicables quedan `SKIPPED` y se excluyen del denominador del progreso general. `Pedido confirmado` es obligatorio y se crea como `COMPLETED`. La primera etapa productiva aplicable se crea como `IN_PROGRESS`.

Solo puede existir una etapa `IN_PROGRESS` o `BLOCKED` por orden. Una orden activa o pausada debe tener `current_stage_id`; una completada o cancelada no.

### 8.5 Historial y avances

#### `order_updates`

- `id UUID PK`
- `workshop_id UUID FK NOT NULL`
- `order_id UUID FK NOT NULL`
- `order_stage_id UUID NULL`
- `user_id UUID FK NOT NULL`
- `event_type VARCHAR NOT NULL`
- `quantity_delta INTEGER NULL`
- `comment TEXT NULL`
- `corrects_update_id UUID NULL`
- `client_request_id UUID NULL`
- `created_at TIMESTAMPTZ NOT NULL`
- Índice único parcial sobre `(order_id, client_request_id)` cuando `client_request_id` no sea nulo

Tipos iniciales:

- `PROGRESS_RECORDED`
- `PROGRESS_CORRECTED`
- `PROBLEM_REPORTED`
- `STAGE_ADVANCED`
- `ORDER_PAUSED`
- `ORDER_RESUMED`
- `ORDER_CANCELLED`
- `ORDER_COMPLETED`

Los avances son incrementales. El acumulado de una etapa es la suma de sus `quantity_delta`. Una corrección crea un nuevo ajuste positivo o negativo y referencia el registro original. Los registros existentes no se reescriben ni se eliminan.

El `client_request_id`, generado por el frontend, evita duplicar avances durante reintentos.

### 8.6 Evidencias

#### `attachments`

- `id UUID PK`
- `workshop_id UUID FK NOT NULL`
- `order_update_id UUID FK NOT NULL`
- `storage_key VARCHAR UNIQUE NOT NULL`
- `mime_type VARCHAR NOT NULL`
- `size_bytes BIGINT NOT NULL`
- `is_client_visible BOOLEAN NOT NULL DEFAULT FALSE`
- `created_by UUID FK NOT NULL`
- `created_at TIMESTAMPTZ NOT NULL`
- `updated_at TIMESTAMPTZ NOT NULL`

Las evidencias nacen como internas. Solo `OWNER` o `ADMIN` pueden cambiar su visibilidad para el cliente.

## 9. Reglas de negocio

### 9.1 Progreso

- El operario registra incrementos, no acumulados manuales.
- Solo se registra avance en la etapa actual.
- La cantidad acumulada no puede superar el objetivo de la etapa.
- Alcanzar el 100 % no cambia automáticamente de etapa.
- `OWNER` o `ADMIN` confirma el avance a la siguiente etapa.
- El cambio completa la etapa actual y activa la siguiente en una sola transacción.
- Al completar la última etapa aplicable, la orden pasa a `COMPLETED`.

### 9.2 Pausa y reanudación

Al pausar:

- La orden pasa de `ACTIVE` a `PAUSED`.
- La etapa actual pasa de `IN_PROGRESS` a `BLOCKED`.
- Se registra motivo, comentario, autor y fecha.

Al reanudar:

- La orden vuelve a `ACTIVE`.
- La etapa vuelve a `IN_PROGRESS`.
- Se añade un evento de reanudación.

Un `OPERATOR` registra `PROBLEM_REPORTED`, pero no cambia directamente el estado de la orden.

### 9.3 Cancelación

- Solo `OWNER` o `ADMIN` puede cancelar.
- Se solicita confirmación explícita.
- La orden pasa a `CANCELLED` y pierde su etapa actual.
- El historial se conserva.
- El tracking deja de estar disponible.

### 9.4 Semáforo de entrega

El estado operativo y el estado de fecha son conceptos separados:

- `LATE`: entrega anterior a hoy y orden no completada.
- `AT_RISK`: entrega hoy o mañana y la orden aún no está en la última etapa.
- `ON_TIME`: cualquier otro caso activo.
- `COMPLETED`: orden finalizada.

Los cálculos usan la fecha local de `America/Lima`. Los timestamps se almacenan en UTC y las fechas comprometidas como `DATE`.

## 10. API REST

### 10.1 Sesión y taller

```http
GET /api/v1/me
GET /api/v1/workshops/current
```

### 10.2 Equipo

```http
GET   /api/v1/team
POST  /api/v1/team/invitations
PATCH /api/v1/team/{membershipId}
```

### 10.3 Clientes

```http
GET   /api/v1/clients
POST  /api/v1/clients
GET   /api/v1/clients/{id}
PATCH /api/v1/clients/{id}
```

### 10.4 Órdenes

```http
GET   /api/v1/orders
POST  /api/v1/orders
GET   /api/v1/orders/{id}
PATCH /api/v1/orders/{id}

POST /api/v1/orders/{id}/pause
POST /api/v1/orders/{id}/resume
POST /api/v1/orders/{id}/cancel
PATCH /api/v1/orders/{id}/tracking
POST  /api/v1/orders/{id}/tracking/regenerate
```

`PATCH /tracking` activa o desactiva el acceso externo. `POST /tracking/regenerate`
reemplaza el token anterior de forma atómica y lo invalida inmediatamente. Ambas
operaciones requieren rol `OWNER` o `ADMIN`.

Filtros del listado:

- `status`
- `client_id`
- `stage`
- `delivery_from`
- `delivery_to`
- `page`
- `page_size`

### 10.5 Producción

```http
GET   /api/v1/orders/{id}/stages
PATCH /api/v1/orders/{id}/stages/{stageId}/assignee
POST  /api/v1/orders/{id}/stages/{stageId}/advance

GET  /api/v1/orders/{id}/updates
POST /api/v1/orders/{id}/updates
POST /api/v1/orders/{id}/updates/{updateId}/corrections
```

### 10.6 Evidencias

```http
POST  /api/v1/orders/{id}/updates/{updateId}/attachments
PATCH /api/v1/orders/{id}/attachments/{attachmentId}/visibility
```

La carga usa `multipart/form-data`, con un máximo inicial de 10 MB y formatos JPEG, PNG o WebP.

### 10.7 Dashboard y tracking

```http
GET /api/v1/dashboard/summary
GET /api/v1/tracking/{token}
```

El tracking no requiere JWT y devuelve un DTO público separado del modelo interno.

## 11. Contrato HTTP

Respuesta exitosa:

```json
{
  "data": {},
  "meta": {}
}
```

Respuesta de error:

```json
{
  "error": {
    "code": "ORDER_NOT_ACTIVE",
    "message": "La orden no admite nuevos avances.",
    "details": {},
    "request_id": "uuid"
  }
}
```

Códigos principales:

- `400`: petición inválida.
- `401`: autenticación ausente o inválida.
- `403`: acción no permitida para el rol.
- `404`: recurso inexistente o perteneciente a otro taller.
- `409`: conflicto de estado, concurrencia o petición duplicada.
- `422`: regla de negocio incumplida.
- `500`: error inesperado.

La API no revela si un UUID pertenece a otro taller.

## 12. Flujos transaccionales

### 12.1 Crear orden

1. Validar JWT, membresía y rol.
2. Validar cliente, fechas, cantidad y etapas.
3. Bloquear el contador del taller y generar el código.
4. Generar el token de tracking.
5. Crear la orden y las siete etapas.
6. Marcar las no aplicables como `SKIPPED`.
7. Completar `Pedido confirmado` y activar la primera etapa aplicable.
8. Confirmar la transacción.

### 12.2 Registrar avance

1. Validar membresía, rol y asignación.
2. Bloquear la orden y la etapa actual.
3. Comprobar que la orden esté activa.
4. Verificar `client_request_id`.
5. Verificar que el nuevo acumulado no supere el objetivo.
6. Insertar el evento incremental.
7. Devolver acumulado, resumen e historial actualizado.

### 12.3 Cambiar de etapa

1. Bloquear orden y etapas.
2. Confirmar que la etapa actual alcanzó el objetivo cuando corresponda.
3. Completar la etapa actual.
4. Activar la siguiente etapa aplicable o completar la orden.
5. Insertar el evento de trazabilidad.
6. Confirmar la transacción.

## 13. Autenticación y seguridad

El frontend obtiene un JWT de Supabase Auth y lo envía como `Authorization: Bearer <token>`. La API valida firma, emisor, audiencia y expiración, resuelve al usuario local y comprueba su membresía.

Toda operación interna sigue esta secuencia:

```text
JWT válido
→ usuario local
→ taller activo
→ membresía activa
→ permiso del rol
→ recurso del mismo taller
→ servicio de aplicación
```

Medidas adicionales:

- CORS limitado a los orígenes configurados.
- Secretos únicamente mediante variables de entorno.
- Consultas parametrizadas mediante GORM.
- Rate limiting básico sobre el tracking público y cargas.
- El token de tracking no aparece en logs.
- Los logs no contienen JWT, contraseñas, fotografías ni datos personales completos.
- Los nombres de archivos no provienen del cliente.
- La validación de imágenes comprueba tamaño, MIME real y decodificación.

Los objetos se guardan con una clave generada por el servidor:

```text
{workshop_id}/{order_id}/{uuid}
```

Si falla la fotografía después de registrar un avance, el avance se conserva y la carga puede reintentarse.

## 14. Concurrencia y consistencia

Las siguientes operaciones usan transacciones y bloqueo de filas en PostgreSQL:

- Registro y corrección de avances.
- Cambio de etapa.
- Pausa y reanudación.
- Cancelación.
- Finalización.
- Generación del código de orden.

Un conflicto devuelve `409`. El frontend recarga el detalle y muestra un mensaje comprensible. El historial productivo no utiliza actualizaciones optimistas.

## 15. Experiencia frontend

### Dueño y administrador

Experiencia principalmente desktop, responsive para móvil:

- Dashboard con órdenes activas, atrasadas y próximas a entregar.
- Listado filtrable.
- Creación rápida de clientes y órdenes.
- Detalle con resumen, etapas, responsable, historial y evidencias.
- Administración de equipo y visibilidad del tracking.

### Operario

Experiencia mobile-first:

- Solo órdenes asignadas.
- Etapa y objetivo claramente visibles.
- Cantidad incremental con accesos rápidos.
- Comentario opcional.
- Evidencia interna opcional.
- Reporte de problema.

### Cliente externo

- Sin cuenta.
- Resumen de la orden.
- Etapa actual y progreso autorizado.
- Timeline simple.
- Última actualización.
- Solo evidencias publicadas.
- Enlace de contacto con el taller.

Estados obligatorios de interfaz:

- Cargando.
- Vacío.
- Sin resultados.
- Error de conexión.
- Validación.
- Conflicto concurrente.
- Sesión vencida.
- Carga de fotografía fallida.
- Permiso insuficiente.

## 16. Flyway y Docker

Migraciones iniciales propuestas:

```text
V1__create_extensions.sql
V2__create_identity_and_workshops.sql
V3__create_clients.sql
V4__create_orders_and_stages.sql
V5__create_updates_and_attachments.sql
V6__add_constraints_and_indexes.sql
```

Reglas:

- Una migración aplicada no se edita.
- Cada cambio de esquema crea una versión nueva.
- Las migraciones son forward-only.
- CI ejecuta `flyway validate` y migra una base vacía.
- Los entornos productivos ejecutan Flyway como tarea previa al arranque de la API.

Docker Compose incluirá:

- `frontend`
- `backend`
- `postgres`
- `flyway`

Supabase Auth y Storage se consumen como servicios externos configurados por variables de entorno.

## 17. Observabilidad

Endpoints:

```http
GET /health/live
GET /health/ready
```

Logs JSON:

- `timestamp`
- `level`
- `request_id`
- `method`
- `path`
- `status`
- `duration`
- `user_id`, cuando corresponda
- `workshop_id`, cuando corresponda
- `error_code`

## 18. Estrategia de pruebas

### Backend

- Pruebas unitarias de servicios y reglas de negocio.
- Pruebas HTTP con `httptest`.
- Pruebas de repositorios contra PostgreSQL real.
- Pruebas de aislamiento entre talleres.
- Pruebas de permisos por rol.
- Pruebas de concurrencia e idempotencia.
- Pruebas del DTO público de tracking.

### Frontend

- Vitest y Vue Test Utils.
- Formularios y validaciones críticas.
- Stores de sesión y taller.
- Manejo de errores de API.
- Visibilidad de acciones según rol.

### End-to-end

Playwright cubrirá al menos:

```text
Login
→ cliente
→ orden
→ avance
→ historial
→ tracking
```

También verificará que un operario no pueda administrar usuarios, corregir avances ni publicar evidencias.

### CI

```text
Backend:
go test ./...
go vet ./...

Frontend:
npm ci
npm run lint
npm run test
npm run build

Database:
flyway validate
flyway migrate sobre PostgreSQL vacío
```

## 19. Criterios de aceptación arquitectónicos

- La API puede ejecutarse como un único servicio Go.
- El frontend solo consume la API para datos de negocio.
- Supabase Auth es la única fuente de autenticación.
- Flyway es la única fuente de cambios de esquema.
- Ninguna consulta interna permite cruzar talleres.
- El historial de producción es append-only.
- Los reintentos no duplican avances.
- Las evidencias son internas por defecto.
- El tracking utiliza un DTO público y un token no enumerable.
- La base distingue estado productivo de semáforo de entrega.
- El sistema funciona sin Redis, WebSockets o procesos asíncronos obligatorios.
- El flujo crítico cuenta con pruebas unitarias, de integración y end-to-end.

## 20. Próximo paso

Con esta especificación aprobada, el siguiente artefacto será un plan de implementación por fases. El plan debe comenzar por el entorno y las migraciones, continuar con un vertical slice completo y postergar cualquier funcionalidad que no sea necesaria para demostrar la trazabilidad de una orden.
