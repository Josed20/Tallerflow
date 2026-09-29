# TallerFlow MVP — Especificación de producto y arquitectura

**Estado:** Aprobada para planificación

**Fecha:** 2026-09-21

**Alcance:** MVP de trazabilidad para talleres textiles

**Sustituye:** arquitectura anterior con Supabase, Vercel y Render

## 1. Propósito y resultado esperado

TallerFlow permite que un taller registre y consulte el recorrido de una orden desde su confirmación hasta la entrega. El operario registra, el dueño o administrador controla y el cliente consulta mediante un enlace privado.

El MVP se dirige inicialmente a talleres pequeños y medianos de Gamarra y Lima, con 10 a 30 operarios y varias órdenes simultáneas. Debe permitir:

1. Iniciar sesión mediante autenticación propia.
2. Crear clientes y órdenes.
3. Configurar etapas y responsables.
4. Registrar avances desde un teléfono.
5. Conservar historial y correcciones.
6. Adjuntar evidencias con visibilidad controlada.
7. Consultar el tracking externo sin crear una cuenta.
8. Pausar, reanudar, cancelar y completar según permisos.

Crear una orden debe tomar menos de cinco minutos y registrar un avance habitual menos de un minuto. Las vistas principales funcionarán desde 360 px hasta escritorio.

## 2. Decisiones definitivas

| Área | Decisión |
|---|---|
| Frontend | Vue 3 SPA, TypeScript, Vite y Composition API |
| Navegación y estado | Vue Router, TanStack Query y Pinia solo para estado global |
| Diseño | Tailwind CSS, Reka UI y sistema visual propio |
| Formularios | VeeValidate y Zod |
| Backend | Go, Gin y GORM como monolito modular |
| API | REST JSON en `/api/v1`, mismo origen que el frontend |
| Autenticación | Go, Argon2id y sesiones opacas en cookies HttpOnly |
| Datos | PostgreSQL y Flyway como única autoridad del esquema |
| Multi-tenancy | `workshop_id`, autorización de aplicación y RLS |
| Archivos | Adaptador privado; volumen persistente y evolución a S3 compatible |
| Producción | Caddy, Docker Compose, dos VPS y backup externo cifrado |

No se utilizarán Supabase, Firebase, Auth0, Clerk, `GORM AutoMigrate`, Kubernetes ni microservicios durante el MVP.

## 3. Fuera de alcance

- Contabilidad, facturación, planillas, POS o inventario avanzado.
- Marketplace, logística, cobros o suscripciones.
- Aplicaciones móviles nativas.
- Integración técnica con Gamarra Mayoristas.
- Trabajo offline, IA o predicción de retrasos.
- WebSockets, Kafka, CQRS o event sourcing.
- Alta disponibilidad automática entre regiones.
- Segundo factor de autenticación en el primer corte.

## 4. Diseño visual

Los diseños aprobados en Stitch son referencia visual, no código. Se implementarán login, dashboard, órdenes, creación y detalle de orden, registro de avances, clientes, equipo, tracking y sus estados de carga, vacío, error y confirmación.

El producto no debe parecer una plantilla administrativa genérica. El sistema visual de TallerFlow definirá color, tipografía, espaciado, radios, sombras, iconografía, estados semánticos y movimiento. Sus elementos distintivos serán:

- Timeline de producción.
- Tarjetas de pedido.
- Indicadores de avance y semáforos de entrega.
- Tablas responsivas.
- Formularios guiados.
- Skeletons, estados vacíos y confirmaciones.

Las animaciones serán breves y funcionales mediante Vue y CSS. Se respetará `prefers-reduced-motion`.

## 5. Arquitectura general

```text
                              INTERNET
                                  |
                       https://app.tallerflow.pe
                                  |
                     +------------v------------+
                     |        Caddy/TLS        |
                     | headers + reverse proxy |
                     +------------+------------+
                                  |
                 +----------------+----------------+
                 |                                 |
       Vue 3 SPA compilada                    /api/v1/*
       archivos estáticos                          |
                 |                       +---------v---------+
                 |                       | API Go            |
                 |                       | Gin + GORM        |
                 |                       | monolito modular  |
                 |                       +---------+---------+
                 +--------- mismo origen ----------+
                                                   |
                                        red privada/WireGuard
                                                   |
                                    +--------------v--------------+
                                    | PostgreSQL + Flyway         |
                                    | pgBackRest + backup externo |
                                    +-----------------------------+
```

Caddy sirve Vue y reenvía `/api/v1/*` a Go. Todo funciona bajo `app.tallerflow.pe`; producción no requiere CORS y las cookies permanecen en el mismo origen.

## 6. Organización del repositorio

```text
Tallerflow/
├── backend/
│   ├── cmd/{api,worker,bootstrap}/
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
│   │   └── audit/
│   ├── platform/{config,database,httpx,security,storage,observability}/
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── app/
│   │   ├── modules/
│   │   ├── components/{ui,shared}/
│   │   ├── services/
│   │   ├── stores/
│   │   ├── styles/
│   │   └── types/
│   └── package.json
├── database/migrations/
├── infra/{caddy,docker,backup}/
├── docs/
├── compose.yaml
└── compose.production.yaml
```

Cada módulo backend contiene dominio, aplicación, puertos y adaptadores. `platform` no contiene reglas de negocio. Cada módulo frontend agrupa sus vistas, componentes, consultas y formularios.

## 7. Frontend

- Vue Router controla rutas públicas, privadas y carga diferida.
- TanStack Query controla datos remotos, caché, reintentos e invalidación.
- Pinia conserva únicamente sesión visible, taller activo y preferencias.
- Reka UI aporta comportamiento accesible sin imponer apariencia.
- Tailwind y tokens CSS implementan la identidad visual.
- VeeValidate y Zod centralizan formularios y mensajes.
- `fetch` usa `credentials: "same-origin"` y CSRF en mutaciones.

No se copiarán respuestas de la API a Pinia. El historial, cambios de etapa, cancelaciones y correcciones no usarán actualizaciones optimistas.

Los tipos de transporte se generarán desde OpenAPI. El frontend debe incluir navegación por teclado, foco visible, contraste WCAG AA en controles esenciales, división de código por ruta y estados comprensibles.

## 8. Backend modular

Cada módulo aplica:

```text
domain/          entidades, estados y reglas puras
application/     casos de uso y transacciones
ports/           interfaces requeridas
adapters/        HTTP, GORM y almacenamiento
```

Responsabilidades:

- `auth`: credenciales, sesiones, CSRF, intentos y recuperación.
- `workshops`: taller activo y configuración.
- `team`: membresías, roles e incorporación.
- `clients`: clientes.
- `orders`: orden, numeración y ciclo de vida.
- `production`: etapas, avances, problemas y correcciones.
- `attachments`: almacenamiento y visibilidad.
- `dashboard`: proyecciones de lectura.
- `tracking`: proyección pública.
- `audit`: acciones de seguridad y administración.

Los módulos se comunican mediante interfaces o casos de uso explícitos. Ningún handler accede directamente a GORM.

## 9. Roles

- `OWNER`: control total.
- `ADMIN`: operación, clientes, órdenes y equipo, salvo transferir propiedad.
- `OPERATOR`: órdenes asignadas y avances.

| Acción | OWNER | ADMIN | OPERATOR |
|---|---:|---:|---:|
| Administrar usuarios | Sí | Sí | No |
| Transferir propiedad | Sí | No | No |
| Crear clientes y órdenes | Sí | Sí | No |
| Asignar responsables | Sí | Sí | No |
| Registrar avances | Sí | Sí | Solo asignadas |
| Corregir y cambiar etapa | Sí | Sí | No |
| Pausar, reanudar o cancelar | Sí | Sí | No |
| Reportar problema | Sí | Sí | Solo asignadas |
| Publicar evidencias | Sí | Sí | No |

El visitante externo no es usuario interno y solo recibe una proyección mediante token.

## 10. Autenticación propia

Tablas principales:

- `users`: identidad, email único, nombre, estado y fechas.
- `user_credentials`: hash Argon2id, fecha de cambio y cambio obligatorio.
- `user_sessions`: hash del token, hash CSRF, usuario, expiración y revocación.
- `login_attempts`: email anonimizado, prefijo IP, resultado y fecha.
- `password_reset_tokens`: hash de token, expiración, uso y creador.

Flujo:

1. La API valida origen, formato y límite de intentos.
2. Verifica la contraseña con Argon2id y parámetros versionados.
3. Genera un token criptográfico de 32 bytes.
4. PostgreSQL almacena solo su hash.
5. El navegador recibe `__Host-tallerflow_session` con `Secure`, `HttpOnly`, `SameSite=Lax` y `Path=/`.
6. La sesión dura como máximo ocho horas.
7. Login, cambio de contraseña o privilegio rotan la sesión.
8. Logout y desactivación revocan sesiones.

No se guardan JWT ni credenciales en almacenamiento web. Toda mutación autenticada requiere `X-CSRF-Token` asociado a la sesión y validación de `Origin` o `Referer`.

El primer `OWNER` se crea mediante `bootstrap`. El MVP no depende de correo: OWNER o ADMIN genera un enlace de incorporación o recuperación de un solo uso para compartir por un canal acordado. La base conserva únicamente su hash.

## 11. Multi-tenancy

`workshops` contiene nombre, contacto, zona horaria y fechas. `workshop_members` relaciona usuario, taller, rol y estado con unicidad por pareja.

Toda tabla privada incluye `workshop_id`. La defensa opera en dos capas:

1. Casos de uso y repositorios filtran por taller y permiso.
2. PostgreSQL aplica Row-Level Security usando el taller establecido mediante `SET LOCAL` dentro de la transacción.

La aplicación usa un rol sin propiedad de tablas ni `BYPASSRLS`. Flyway usa otro rol, privilegiado solo durante migraciones. Sin contexto válido, las políticas deniegan acceso.

## 12. Modelo operativo

### Clientes

`clients` contiene ID, taller, nombre, empresa, teléfono, email, estado activo y fechas. No existe eliminación física durante el MVP.

### Órdenes

`workshop_counters` genera códigos secuenciales por taller con bloqueo de fila.

`orders` contiene:

- Taller, cliente y código único por taller.
- Producto, descripción y cantidad positiva.
- Fechas de inicio y entrega coherentes.
- Estado, etapa actual y responsable.
- `tracking_token_hash` y `tracking_enabled`.
- Creador y timestamps.

Estados: `ACTIVE`, `PAUSED`, `COMPLETED` y `CANCELLED`.

El token de tracking contiene 32 bytes aleatorios codificados para URL. Solo se entrega al crear o regenerar el enlace; la base conserva su hash. No contiene datos del cliente ni aparece en logs.

### Etapas

`order_stages` contiene taller, orden, nombre, secuencia, estado, objetivo, responsable y fechas.

Estados: `PENDING`, `IN_PROGRESS`, `COMPLETED`, `BLOCKED` y `SKIPPED`.

Plantilla inicial:

1. Pedido confirmado.
2. Corte.
3. Confección.
4. Estampado o bordado.
5. Acabado.
6. Control de calidad.
7. Entrega.

Las no aplicables quedan `SKIPPED`. Pedido confirmado nace `COMPLETED`; la primera etapa productiva aplicable, `IN_PROGRESS`. Solo puede existir una etapa `IN_PROGRESS` o `BLOCKED` por orden.

### Avances

`order_updates` contiene taller, orden, etapa, usuario, tipo, delta, comentario, corrección, `client_request_id` y fecha.

Tipos: `PROGRESS_RECORDED`, `PROGRESS_CORRECTED`, `PROBLEM_REPORTED`, `STAGE_ADVANCED`, `ORDER_PAUSED`, `ORDER_RESUMED`, `ORDER_CANCELLED` y `ORDER_COMPLETED`.

Los avances son incrementales. Corregir añade un ajuste y referencia el original; no edita ni elimina eventos. `client_request_id` evita duplicados.

### Evidencias

`attachments` contiene taller, actualización, clave, nombre original, MIME, tamaño, SHA-256, visibilidad, creador y fecha.

Las evidencias nacen privadas. Solo OWNER o ADMIN puede publicarlas. Se aceptan JPEG, PNG y WebP hasta 10 MB; la API verifica tamaño, firma real y decodificación.

## 13. Reglas de negocio

- El operario registra incrementos solo en la etapa actual.
- El acumulado no puede superar el objetivo.
- Llegar al 100 % no cambia automáticamente de etapa.
- OWNER o ADMIN confirma el cambio en una transacción.
- Completar la última etapa completa la orden.
- Pausar cambia la orden a `PAUSED` y la etapa a `BLOCKED`.
- OPERATOR reporta un problema, pero no pausa directamente.
- Cancelar exige rol, confirmación y motivo; conserva historial y desactiva tracking.

Semáforo:

- `LATE`: entrega anterior a hoy y no completada.
- `AT_RISK`: entrega hoy o mañana y no está en la última etapa.
- `ON_TIME`: otro caso activo.
- `COMPLETED`: finalizada.

Timestamps en UTC; fechas comprometidas como `DATE` interpretadas en `America/Lima`.

## 14. API REST

```http
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/session
POST /api/v1/auth/change-password
POST /api/v1/auth/reset/consume
GET  /api/v1/me

GET   /api/v1/workshops/current
GET   /api/v1/team
POST  /api/v1/team/invitations
PATCH /api/v1/team/{membershipId}
POST  /api/v1/team/{membershipId}/reset-link

GET   /api/v1/clients
POST  /api/v1/clients
GET   /api/v1/clients/{id}
PATCH /api/v1/clients/{id}

GET   /api/v1/orders
POST  /api/v1/orders
GET   /api/v1/orders/{id}
PATCH /api/v1/orders/{id}
POST  /api/v1/orders/{id}/pause
POST  /api/v1/orders/{id}/resume
POST  /api/v1/orders/{id}/cancel
PATCH /api/v1/orders/{id}/tracking
POST  /api/v1/orders/{id}/tracking/regenerate

GET   /api/v1/orders/{id}/stages
PATCH /api/v1/orders/{id}/stages/{stageId}/assignee
POST  /api/v1/orders/{id}/stages/{stageId}/advance
GET   /api/v1/orders/{id}/updates
POST  /api/v1/orders/{id}/updates
POST  /api/v1/orders/{id}/updates/{updateId}/corrections
POST  /api/v1/orders/{id}/updates/{updateId}/attachments
GET   /api/v1/orders/{id}/attachments/{attachmentId}/content
PATCH /api/v1/orders/{id}/attachments/{attachmentId}/visibility

GET /api/v1/dashboard/summary
GET /api/v1/tracking/{token}
GET /api/v1/tracking/{token}/attachments/{attachmentId}
```

Tracking devuelve un DTO separado, sin IDs internos, contactos privados, responsables, comentarios privados ni evidencias no publicadas.

## 15. Contrato HTTP

Éxito:

```json
{"data": {}, "meta": {}}
```

Error:

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

- `400`: petición inválida.
- `401`: sesión ausente o vencida.
- `403`: acción no permitida.
- `404`: recurso inexistente o de otro taller.
- `409`: conflicto o duplicado.
- `422`: regla de negocio.
- `429`: límite temporal.
- `500`: error inesperado.

La API asigna `request_id` y no revela si un UUID pertenece a otro taller. OpenAPI es la referencia compartida del transporte; Flyway es la única referencia del esquema físico.

## 16. Transacciones e idempotencia

Usan transacción y bloqueo de filas:

- Numeración y creación de orden.
- Registro y corrección de avances.
- Cambio de etapa.
- Pausa, reanudación, cancelación y finalización.
- Regeneración de tracking.
- Consumo de tokens de incorporación o recuperación.

Un conflicto devuelve `409`. El frontend recarga la consulta y explica el cambio. Las mutaciones con riesgo de reenvío usan `client_request_id` o clave de idempotencia.

## 17. Archivos

El dominio depende de `ObjectStorage`, no de rutas físicas. El adaptador inicial guarda objetos en un volumen persistente fuera del contenedor con claves:

```text
{workshop_id}/{order_id}/{uuid}
```

PostgreSQL almacena metadatos, no binarios. Go verifica sesión, taller y permiso antes de entregar archivos. Para tracking también valida token y visibilidad.

Migrar a S3 compatible reemplazará el adaptador sin cambiar órdenes o producción. Una carga fallida no revierte un avance confirmado.

## 18. Flyway y code-first

Go define comportamiento y casos de uso. El esquema se versiona como SQL:

```text
V1__create_extensions.sql
V2__create_identity_sessions_and_workshops.sql
V3__create_clients.sql
V4__create_orders_and_stages.sql
V5__create_updates_and_attachments.sql
V6__add_constraints_indexes_and_rls.sql
V7__create_audit_log.sql
```

- Una migración aplicada nunca se edita.
- Cada cambio crea una versión forward-only.
- No existe `AutoMigrate`.
- CI ejecuta `flyway validate` y migra una base vacía.
- Producción ejecuta Flyway antes de la nueva API.
- GORM se prueba contra PostgreSQL real, no SQLite.
- Flyway y la aplicación usan roles distintos.

## 19. Infraestructura y dominio

### Dominio y borde

- Dominio: `app.tallerflow.pe`.
- DNS independiente del proveedor de VPS.
- TLS automático con Caddy.
- HSTS, CSP, `X-Content-Type-Options`, política de referrer y límites de cuerpo.
- Solo puertos 80/443 públicos en aplicación.
- SSH con llaves, sin contraseña y acceso restringido.

### VPS 1 — Aplicación

- Caddy.
- Vue compilado.
- API Go.
- Worker para limpiar sesiones, tokens y archivos huérfanos.
- Docker Compose de producción.
- Volumen persistente de evidencias durante el MVP.
- Logs JSON con rotación.

### VPS 2 — Datos

- PostgreSQL.
- pgBackRest.
- Sin puerto PostgreSQL público.
- Acceso por red privada o WireGuard.
- Roles separados para migración, aplicación y backup.

### Backup externo

- Fuera de ambos VPS y cifrado antes de salir.
- WAL continuo, backup completo semanal e incrementales diarios.
- Evidencias copiadas al menos cada hora.
- Retención inicial: 7 diarios, 4 semanales y 3 mensuales.
- Restauración de prueba mensual.

Objetivos iniciales:

- RPO de base de datos: 15 minutos.
- RPO de evidencias: 1 hora.
- RTO manual: 4 horas.

Dos VPS separan aplicación y datos, pero no son alta disponibilidad. Réplica PostgreSQL y segundo nodo de aplicación quedan para una fase basada en métricas.

## 20. Seguridad operativa

- Secretos fuera del repositorio.
- Contenedores sin root cuando sea posible.
- Versiones de imágenes fijadas.
- Rate limiting en login, recuperación, tracking y cargas.
- Bloqueo progresivo sin denegación permanente provocable por terceros.
- Auditoría de login, roles, desactivación, tracking, visibilidad y acciones destructivas.
- Logs sin contraseñas, cookies, tokens, fotografías ni datos personales completos.
- Consultas parametrizadas y validación estricta.
- Dependencias analizadas en CI.
- Mínimo privilegio en base de datos, sistema y despliegue.

## 21. Observabilidad

```http
GET /health/live
GET /health/ready
```

Los logs JSON incluirán nivel, fecha, servicio, request ID, ruta, estado y latencia. `/health/ready` verifica dependencias sin exponer secretos.

El MVP empieza con logs, health checks, métricas básicas y alerta externa. Prometheus, Grafana, trazas y Valkey se incorporarán solo si la operación los justifica.

## 22. Pruebas

Backend:

- Reglas unitarias.
- Casos de uso con puertos simulados.
- Repositorios sobre PostgreSQL real migrado.
- HTTP, sesiones, CSRF, permisos y errores.
- Concurrencia en numeración y avances.
- `go test -race` y `go vet`.

Frontend:

- Vitest y Vue Test Utils.
- Simulación HTTP en pruebas de componentes.
- Playwright para login, cliente, orden, avance y tracking.
- Accesibilidad automatizada en vistas críticas.
- `vue-tsc`, lint, pruebas y build.

Datos e infraestructura:

- Migración desde base vacía y `flyway validate`.
- Constraints, índices y RLS.
- Restauración en entorno aislado.
- Compose y health checks.

Un flujo no termina si solo funciona con mocks.

## 23. Fases

### Fase 1 — Cimientos seguros

Repositorio, Docker, PostgreSQL, Flyway, backend base, Vue + TypeScript, sistema visual, sesiones, CSRF, talleres, roles y RLS. Resultado: un `OWNER` inicia sesión de forma segura.

### Fase 2 — Operación básica

Clientes, órdenes, numeración, etapas, dashboard y listados responsive. Resultado: el taller crea y consulta una orden.

### Fase 3 — Producción trazable

Asignaciones, avances, correcciones, problemas, cambios de estado, historial y auditoría. Resultado: el operario registra y el dueño controla.

### Fase 4 — Cliente y evidencias

Carga segura, visibilidad, tracking y regeneración de enlaces. Resultado: el cliente consulta sin cuenta y solo ve lo autorizado.

### Fase 5 — Producción

Caddy, dominio, dos VPS, backups, restauración, endurecimiento, E2E y manual operativo. Resultado: piloto desplegado y recuperable.

## 24. Evolución

1. Medir tráfico, latencia, concurrencia y crecimiento.
2. Añadir otra instancia Go cuando sea necesario.
3. Incorporar Valkey solo para caché, rate limiting distribuido o sesiones si PostgreSQL deja de ser suficiente.
4. Migrar archivos a S3 compatible.
5. Añadir segundo nodo de aplicación.
6. Añadir réplica PostgreSQL y failover.
7. Extraer microservicios solo ante una necesidad operativa demostrada.

## 25. Precedencia documental

Esta especificación sustituye decisiones anteriores que indiquen:

- Supabase Auth o Storage.
- Vercel o Render como arquitectura productiva.
- JWT administrado por el frontend.
- JavaScript sin TypeScript.
- Axios obligatorio.
- ausencia de RLS o autenticación propia.

El plan y Sprint 1 anteriores deben regenerarse después de aprobar esta especificación. Si un documento anterior contradice este archivo, prevalece este archivo y no debe iniciarse ese trabajo.

## 26. Definición de terminado

1. Un `OWNER` inicia sesión mediante autenticación propia.
2. Administra miembros, clientes y órdenes dentro de su taller.
3. Un `OPERATOR` solo accede a lo asignado y registra desde móvil.
4. Etapas y cambios conservan historial auditable.
5. Otro taller no obtiene datos por API ni mediante el rol de aplicación en PostgreSQL.
6. El cliente abre un enlace privado y ve solo la proyección autorizada.
7. Evidencias respetan visibilidad y permisos.
8. Frontend, backend y migraciones pasan CI.
9. Producción usa HTTPS, cookies seguras, CSRF, rate limiting y secretos externos.
10. Existe backup externo cifrado y una restauración demostrada.
11. El piloto corre en dos VPS sin depender de un BaaS.
12. No existe `AutoMigrate`, JWT del navegador, Supabase ni secretos reales en el repositorio.

## 27. Decisiones pendientes fuera del diseño

La selección de proveedor de dominio, proveedor de VPS y destino concreto de backups se realizará antes de la Fase 5 comparando región, soporte, snapshots, red privada y costo. Esas elecciones no cambian las interfaces de la aplicación.
