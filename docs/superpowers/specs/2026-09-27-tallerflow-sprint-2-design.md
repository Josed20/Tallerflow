# TallerFlow — Sprint 2: acceso autónomo y operación básica

- **Fecha:** 2026-09-27
- **Duración propuesta:** 10 días hábiles
- **Equipo:** José, Lucero, Michelle y Stephano
- **Estado:** especificación lista para revisión del equipo
- **Objetivo:** un dueño nuevo crea el primer taller sin terminal, entra a TallerFlow, registra clientes y órdenes, consulta su operación y administra a su equipo.

## 1. Resultado del Sprint

Al terminar el Sprint 2 debe existir una demo autónoma y repetible desde una base vacía:

1. El primer dueño abre TallerFlow y crea el taller y la cuenta OWNER.
2. Inicia sesión y entra al espacio autenticado.
3. Crea un cliente y una orden con sus etapas iniciales.
4. Consulta la lista, el detalle, el semáforo y el dashboard de órdenes.
5. Genera una invitación para incorporar a un miembro del equipo.
6. Puede recuperar la contraseña mediante un enlace temporal.
7. Una ruta inexistente muestra un estado 404 y no se confunde con un problema de autenticación.

El Sprint no se considera terminado si la demo requiere ejecutar manualmente el binario de bootstrap, insertar datos con SQL o editar cookies desde el navegador.

## 2. Evidencia y decisiones previas

Fuentes de verdad:

- Producto y arquitectura: `docs/superpowers/specs/2026-09-21-tallerflow-mvp-design.md`.
- Contrato de identidad: `docs/contracts/identity-persistence.md`.
- Contrato HTTP actual: `docs/contracts/openapi.yaml`.
- Cierre técnico del Sprint 1: `docs/superpowers/plans/2026-09-26-sprint-1-final-integration.md`.
- Hallazgos de usuario incorporados en esta especificación: falta de onboarding, recuperación, 404 real y prevención de doble envío.

Decisiones preservadas:

- PostgreSQL y Flyway son las fuentes del esquema; `AutoMigrate` continúa prohibido.
- Las sesiones siguen siendo opacas, same-origin y almacenadas en cookies seguras.
- El frontend no almacena tokens en `localStorage` ni `sessionStorage`.
- Toda información de negocio se aísla por `workshop_id` y RLS.
- OpenAPI define el transporte compartido antes de integrar handlers y clientes.
- El onboarding de este Sprint crea únicamente el primer taller y OWNER de una instalación vacía. El registro público de múltiples talleres queda fuera de alcance.
- La contraseña elegida en el onboarding web es definitiva: se guarda con `must_change_password=false`, registra `password_changed_at` y crea una sesión opaca para entrar a `/app`. El bootstrap por terminal conserva su contraseña temporal y cambio obligatorio.
- Una recuperación válida revoca todas las sesiones previas y devuelve al login; no inicia sesión automáticamente.
- Un usuario no puede pertenecer a más de un taller durante este Sprint. Una invitación para un correo asociado a otro taller falla sin revelar información del otro taller.
- El módulo de órdenes se implementa como primera versión productiva acotada, no como prototipo desechable.

## 3. Alcance y exclusiones

### Incluye

- Onboarding web del primer taller y OWNER.
- Login, restauración de sesión y cambio obligatorio de contraseña ya existentes.
- Solicitud y consumo de recuperación de contraseña.
- Página 404, estados de carga y prevención de doble envío.
- Clientes, órdenes, etapas iniciales, semáforo y dashboard.
- Gestión básica de miembros e invitaciones.
- OpenAPI, Flyway, RLS, auditoría, pruebas y recorrido E2E.
- Experiencia responsive a 360 px y escritorio.

### No incluye

- Registro público de múltiples talleres en una misma instalación.
- QR, tracking público, fotos, adjuntos o evidencias.
- Avances incrementales de producción, incidencias, pausas o cancelaciones.
- Pagos, facturación, inventario, notificaciones push o aplicación móvil nativa.
- Asignación de órdenes a operarios.
- Métricas comerciales sin una línea base real.

## 4. Regla de independencia entre ramas

Las cuatro ramas parten del mismo `main` y ninguna consume código sin fusionar de otra rama. Cada módulo se prueba con una base aislada y un OWNER de fixture.

Ramas:

| Persona | Rama |
|---|---|
| José | `feat/s2-jose-onboarding-integration` |
| Lucero | `feat/s2-lucero-recovery-routing` |
| Michelle | `feat/s2-michelle-orders` |
| Stephano | `feat/s2-Stephano-team` |

Límites compartidos:

- Cada módulo exporta sus rutas; solo José realiza el cableado final en `backend/cmd/api/app.go` y `frontend/src/app/router.ts` durante integración.
- Cada persona entrega un fragmento OpenAPI autocontenido en `docs/contracts/openapi/sprint2/{onboarding|recovery|orders|team}.yaml`, con `operationId` y nombres de esquema prefijados por módulo, y lo valida en su rama junto con pruebas de contrato HTTP. José consolida el archivo único sin reinterpretar los contratos aprobados ni obligar a una rama a consumir otra.
- Las migraciones se reservan antes de abrir ramas: Michelle `V3` y `V4`, Stephano `V5`. Lucero reutiliza la tabla `password_reset_tokens` creada por V1 y no modifica una migración aplicada. Nadie reutiliza ni renombra una versión aplicada.
- Cada rama usa un proyecto Compose o volumen distinto. No se comparte una base migrada parcialmente.
- Los módulos no importan implementaciones internas de otro dominio. Se comunican mediante principal autenticado, interfaces y DTO documentados.
- Los cambios en archivos compartidos se mantienen mínimos y se integran al final; no son requisito para desarrollar o probar el módulo aislado.

## 5. Contratos congelados antes de desarrollar

### Rutas y módulos

| Módulo | Frontend | API |
|---|---|---|
| Onboarding | `/onboarding` | `GET /api/v1/onboarding/status`, `POST /api/v1/onboarding/workshop` |
| Recuperación | `/forgot-password`, `/reset-password` | `POST /api/v1/auth/password-resets`, `POST /api/v1/auth/password-resets/consume` |
| Clientes | `/app/clients` | `GET/POST /api/v1/clients`, `GET/PATCH /api/v1/clients/{id}` |
| Órdenes | `/app/orders`, `/app/orders/new`, `/app/orders/{id}` | `GET/POST /api/v1/orders`, `GET/PATCH /api/v1/orders/{id}`, etapas y dashboard |
| Equipo | `/app/team`, `/join` | `GET /api/v1/team`, `POST /api/v1/team/invitations`, `POST /api/v1/team/invitations/consume`, `PATCH /api/v1/team/{membershipId}` |

### Convenciones HTTP

- Éxito: `{"data": {}, "meta": {}}`.
- Error: `{"error":{"code":"...","message":"...","details":{},"request_id":"uuid"}}`.
- `401`: sesión ausente o vencida.
- `403`: usuario autenticado sin permiso.
- `404`: recurso inexistente o perteneciente a otro taller.
- `409`: conflicto, repetición incompatible o versión obsoleta.
- `422`: regla de negocio o validación semántica.
- `429`: límite temporal.
- Toda mutación same-origin exige CSRF, salvo los endpoints públicos documentados de onboarding, recuperación y consumo de invitaciones. El consumo de invitaciones se protege con token aleatorio de un solo uso, validación estricta de `Origin`/`Host`, rate limit y respuestas anti-enumeración; no depende de una sesión previa para obtener CSRF.
- Los endpoints públicos tienen rate limit por IP y respuestas que no revelan existencia de usuarios o recursos.

## 6. José — onboarding, contratos e integración

**Objetivo individual:** eliminar la dependencia del bootstrap por terminal y entregar el recorrido final integrado.

**Propiedad principal:**

```text
backend/internal/onboarding/
frontend/src/modules/onboarding/
backend/cmd/api/                 # solo composición final
frontend/src/app/router.ts       # solo composición final
docs/contracts/openapi.yaml      # consolidación final
scripts/verify-sprint2.*
```

### Tareas

- [ ] `J-01` Definir y validar `docs/contracts/openapi/sprint2/onboarding.yaml`, incluidos DTO y códigos de onboarding; documentar el comando común con el que los otros módulos validan sus fragmentos autocontenidos.
- [ ] `J-02` Implementar consulta pública de disponibilidad sin revelar datos del taller o OWNER.
- [ ] `J-03` Implementar creación transaccional del primer taller, usuario, credencial, membresía OWNER y evento de auditoría mediante el repositorio restringido de bootstrap.
- [ ] `J-04` Mantener separadas las conexiones `tallerflow_app` y `tallerflow_bootstrap`; la API normal no recibe permisos elevados y el endpoint público deja de admitir creación después de reclamar `bootstrap_state`.
- [ ] `J-05` Reutilizar el hasher Argon2id y las reglas del contrato de identidad; no duplicar seguridad.
- [ ] `J-06` Impedir condiciones de carrera: dos solicitudes simultáneas producen un solo OWNER y una respuesta segura para la perdedora.
- [ ] `J-07` Rechazar onboarding cuando la instalación ya fue reclamada.
- [ ] `J-08` Crear `/onboarding` con nombre del taller, nombre del dueño, correo, contraseña y confirmación.
- [ ] `J-09` Mostrar requisitos de contraseña, errores por campo, estado ocupado y resultado accesible.
- [ ] `J-10` Crear una sesión opaca después del onboarding y dirigir al nuevo OWNER a `/app` sin marcar cambio obligatorio de la contraseña que acaba de elegir.
- [ ] `J-11` Redirigir una instalación vacía desde `/login` hacia onboarding y una instalación reclamada desde `/onboarding` hacia login.
- [ ] `J-12` Consolidar rutas backend, frontend y OpenAPI de las cuatro ramas sin cambiar sus contratos.
- [ ] `J-13` Crear `scripts/verify-sprint2.ps1` y `.sh` para ejecutar la demo desde volúmenes vacíos y actualizar `.github/workflows/ci.yml` para ejecutarlos con PostgreSQL, correo de prueba, fixtures y evidencias del recorrido de Sprint 2.
- [ ] `J-14` Ejecutar el recorrido final E2E y registrar evidencia de integración.

### Pruebas obligatorias

- Caso feliz desde base vacía.
- Correo inválido, contraseña débil y campos vacíos.
- Segundo OWNER rechazado.
- Dos solicitudes concurrentes producen exactamente un OWNER.
- Fallo intermedio revierte usuario, taller, credencial y membresía.
- Rate limit y respuesta sin información sensible.
- Teclado y viewport de 360 px.

### Criterio de aceptación

Una persona sin conocimientos técnicos abre una instalación vacía, crea el taller, entra al producto y no necesita ejecutar comandos de bootstrap.

## 7. Lucero — recuperación, rutas y estados de autenticación

**Objetivo individual:** recuperar el acceso sin soporte técnico y corregir la experiencia de navegación anónima.

**Propiedad principal:**

```text
backend/internal/passwordreset/
frontend/src/modules/password-reset/
frontend/src/modules/not-found/
database/tests/password_reset_*.sql
infra/mail/
compose.mail.yaml
```

### Tareas

- [ ] `L-01` Definir y validar `docs/contracts/openapi/sprint2/recovery.yaml` para solicitud y consumo de recuperación con respuestas anti-enumeración.
- [ ] `L-02` Reutilizar sin modificar la tabla `password_reset_tokens` creada por V1; verificar sus constraints, índice, grants y aislamiento con pruebas SQL, y documentar cualquier extensión aditiva realmente necesaria para una migración futura coordinada.
- [ ] `L-03` Implementar generación criptográfica y almacenar únicamente el hash.
- [ ] `L-04` Implementar solicitud que responde igual exista o no el correo.
- [ ] `L-05` Entregar el enlace mediante una interfaz `PasswordResetDelivery`; usar Mailpit mediante `compose.mail.yaml` para desarrollo y un adaptador SMTP configurable para producción, sin credenciales en el repositorio ni tokens en logs.
- [ ] `L-06` Consumir el token una sola vez, actualizar Argon2id y revocar todas las sesiones anteriores.
- [ ] `L-07` Crear `/forgot-password` y `/reset-password` con estados enviado, inválido, vencido, consumido y error recuperable.
- [ ] `L-08` Añadir el enlace “Olvidé mi contraseña” al login.
- [ ] `L-09` Implementar una página 404 real y hacer que el guard distinga ruta desconocida de ruta protegida.
- [ ] `L-10` Deshabilitar envíos repetidos mientras login o recuperación están pendientes.
- [ ] `L-11` Mantener mensajes por campo, `aria-live`, foco del primer error y controles de 44 px.
- [ ] `L-12` Añadir pruebas unitarias y E2E para solicitud, consumo, expiración, repetición y navegación.

### Pruebas obligatorias

- Correo existente e inexistente producen la misma respuesta pública.
- Token crudo nunca aparece en PostgreSQL o logs.
- Token vencido, alterado o consumido falla sin cambiar la contraseña.
- Recuperación válida rota credencial y revoca sesiones previas.
- Doble clic produce una sola solicitud.
- Ruta inexistente presenta 404; `/app` sin sesión presenta login.

### Criterio de aceptación

Un usuario recupera el acceso con un enlace temporal y una URL errónea nunca se presenta como problema de sesión.

## 8. Michelle — clientes, órdenes y dashboard

**Objetivo individual:** entregar la primera operación productiva visible de TallerFlow.

**Base funcional vinculante:** secciones 12–16 de `docs/superpowers/specs/2026-09-21-tallerflow-mvp-design.md` y las decisiones acotadas de esta especificación.

**Propiedad principal:**

```text
backend/internal/clients/
backend/internal/orders/
backend/internal/dashboard/
frontend/src/modules/clients/
frontend/src/modules/orders/
frontend/src/modules/dashboard/
database/migrations/V3__clients.sql
database/migrations/V4__orders_and_stages.sql
```

### Tareas

- [ ] `M-01` Definir y validar `docs/contracts/openapi/sprint2/orders.yaml` con clientes, órdenes, etapas, cursor, ETag, idempotencia y dashboard.
- [ ] `M-02` Crear clientes con estado activo/inactivo, sin eliminación física y con RLS.
- [ ] `M-03` Crear contador y código secuencial único por taller bajo bloqueo transaccional.
- [ ] `M-04` Crear órdenes `ACTIVE` con cliente, producto, especificación, cantidad, unidad y fechas coherentes.
- [ ] `M-05` Crear la instantánea de siete etapas; primera aplicable en progreso y no aplicables omitidas.
- [ ] `M-06` Persistir idempotencia para evitar órdenes o códigos duplicados.
- [ ] `M-07` Implementar listas con cursor, búsqueda, filtros y orden estable.
- [ ] `M-08` Implementar detalle y edición con ETag e `If-Match`; conflicto obsoleto devuelve `409`.
- [ ] `M-09` Calcular semáforo `LATE`, `AT_RISK`, `ON_TIME` y `COMPLETED` con reloj inyectable y zona del taller.
- [ ] `M-10` Implementar dashboard de OWNER/ADMIN con atrasadas, en riesgo y recientes.
- [ ] `M-11` Permitir una transición administrativa auditada para confirmar la etapa actual.
- [ ] `M-12` Crear UI de clientes, formulario guiado, lista, filtros, detalle y dashboard.
- [ ] `M-13` Cubrir loading, vacío, error, conflicto, permisos y responsive a 360 px.
- [ ] `M-14` Añadir pruebas de dominio, PostgreSQL/RLS, concurrencia, HTTP, Vue y E2E.

### Pruebas obligatorias

- OWNER/ADMIN crea y edita; OPERATOR recibe `403`.
- Recurso ajeno o inexistente produce el mismo `404`.
- Creación concurrente no repite código.
- Misma clave idempotente devuelve la misma orden; payload diferente devuelve `409`.
- ETag obsoleto no pierde el borrador local.
- Filtros permanecen al volver del detalle.
- Lista y formulario funcionan con teclado y a 360 px.

### Criterio de aceptación

OWNER o ADMIN crea cliente y orden, consulta el semáforo y detalle, y confirma una etapa sin duplicados ni fuga entre talleres.

## 9. Stephano — miembros, invitaciones y permisos

**Objetivo individual:** permitir que el OWNER administre el equipo sin compartir credenciales.

**Propiedad principal:**

```text
backend/internal/team/
frontend/src/modules/team/
frontend/src/modules/invitations/
database/migrations/V5__team_invitations.sql
database/tests/team_*.sql
```

### Tareas

- [ ] `S-01` Definir y validar `docs/contracts/openapi/sprint2/team.yaml` con lista de miembros, invitación, aceptación, contraseña inicial para usuario nuevo y cambio de membresía.
- [ ] `S-02` Crear en V5 `team_invitations` con hash de token, taller, correo normalizado, rol, expiración, creador y consumo.
- [ ] `S-03` Añadir constraints, índices, grants y RLS; imponer en PostgreSQL una sola fila de `workshop_members` por `user_id` durante este Sprint para impedir pertenencia a dos talleres incluso bajo concurrencia.
- [ ] `S-04` Implementar lista de miembros del taller actual sin exponer otros talleres.
- [ ] `S-05` Permitir que OWNER invite `ADMIN` u `OPERATOR`; ADMIN solo puede invitar `OPERATOR`.
- [ ] `S-06` Generar un enlace de invitación mostrado una sola vez; almacenar únicamente el hash.
- [ ] `S-07` Implementar la aceptación como endpoint público transaccional: para un usuario nuevo solicita nombre y contraseña, reutiliza el hasher Argon2id y crea usuario, credencial y membresía en una sola transacción; para una membresía inactiva del mismo taller la reactiva sin cambiar la credencial. Exigir token de un solo uso, validar `Origin`/`Host`, aplicar rate limit, serializar por correo/usuario y rechazar sin enumeración un correo que ya pertenece a otro taller. No depende del módulo de recuperación de Lucero.
- [ ] `S-08` Impedir invitaciones duplicadas activas y membresías duplicadas.
- [ ] `S-09` Permitir activar o desactivar miembros sin eliminación física.
- [ ] `S-10` Impedir que se desactive o degrade al último OWNER activo mediante bloqueo transaccional de las membresías OWNER o una escritura SQL condicional que preserve el invariante bajo solicitudes concurrentes.
- [ ] `S-11` Crear `/app/team` y `/join` con lista, invitación, copia de enlace, estados vacío/error y confirmaciones.
- [ ] `S-12` Añadir pruebas SQL, Go, Vue y E2E de roles, expiración, repetición y aislamiento.

### Pruebas obligatorias

- OWNER y ADMIN ven solo miembros del taller actual.
- ADMIN no crea otro ADMIN ni OWNER.
- Token crudo no aparece en base o logs y solo se consume una vez.
- Invitación vencida o alterada no crea usuario ni membresía.
- Dos aceptaciones concurrentes para el mismo correo desde talleres distintos producen como máximo una membresía y nunca dejan un usuario sin credencial.
- No se puede eliminar el último OWNER activo.
- Dos degradaciones o desactivaciones concurrentes nunca dejan al taller sin OWNER activo.
- Otro taller recibe `404` neutro.
- Vista de equipo funciona con teclado y a 360 px.

### Criterio de aceptación

El OWNER incorpora y administra miembros sin compartir contraseñas y sin romper el aislamiento entre talleres.

## 10. Oleadas del Sprint

### Días 1–2 — contratos y esqueletos

| Persona | Resultado verificable |
|---|---|
| José | OpenAPI base de onboarding y fixture de OWNER |
| Lucero | pruebas de la tabla V1 existente y contratos de recuperación/404 |
| Michelle | V3/V4 y contratos de clientes/órdenes |
| Stephano | V5 y contratos de equipo/invitaciones |

Puerta: los cuatro módulos compilan y sus migraciones funcionan desde una base V1/V2 aislada.

### Días 3–5 — dominio, datos y API

| Persona | Resultado verificable |
|---|---|
| José | primer OWNER transaccional y protegido contra carrera |
| Lucero | token de recuperación, consumo y revocación de sesiones |
| Michelle | clientes y creación idempotente de órdenes |
| Stephano | invitaciones, membresías y reglas del último OWNER |

Puerta: pruebas Go y PostgreSQL reales cubren RLS, roles, errores y concurrencia.

### Días 6–8 — experiencia de usuario

| Persona | Resultado verificable |
|---|---|
| José | onboarding responsive |
| Lucero | recuperación, 404 y estados ocupados |
| Michelle | clientes, órdenes, detalle y dashboard |
| Stephano | equipo y aceptación de invitaciones |

Puerta: pruebas Vue, accesibilidad y E2E del módulo pasan de forma aislada.

### Días 9–10 — integración y demo

- Fusionar las ramas en una rama de integración creada desde `main` actualizado.
- Aplicar V3, V4 y V5 desde una base vacía y ejecutar `flyway validate`.
- Consolidar OpenAPI, rutas y navegación.
- Ejecutar backend, frontend, SQL, Playwright y verificación Compose.
- Repetir la demo completa en escritorio y 360 px.
- Corregir todo defecto crítico o alto antes de publicar.
- Publicar `main` solo con CI verde y evidencia de la demo.

## 11. Estrategia de Git, PR y revisión

Cada persona entrega commits pequeños y PR independientes. No se trabaja directamente en `main`.

PR sugeridos:

| Persona | PR principal |
|---|---|
| José | `feat(onboarding): provision first workshop owner from web` |
| Lucero | `feat(auth): add password recovery and route states` |
| Michelle | `feat(orders): add productive client and order workflow` |
| Stephano | `feat(team): add secure workshop invitations` |

Revisión cruzada:

| Autor | Revisor principal | Foco |
|---|---|---|
| José | Stephano | transacción, secretos y reproducibilidad |
| Lucero | José | auth, sesiones, errores y accesibilidad |
| Michelle | Stephano | SQL, concurrencia, RLS e idempotencia |
| Stephano | Michelle | roles, flujo de usuario y aislamiento |

Checklist de cada PR:

- [ ] Parte de `main` actualizado.
- [ ] Solo modifica el módulo asignado y cambios compartidos mínimos.
- [ ] Incluye pruebas antes o junto con la implementación.
- [ ] No contiene secretos, tokens crudos ni PII en logs.
- [ ] Actualiza su contrato OpenAPI y documentación.
- [ ] Prueba PostgreSQL con el rol real de aplicación y RLS.
- [ ] Prueba loading, vacío, error, permiso y reintento.
- [ ] Funciona con teclado y a 360 px cuando tiene UI.
- [ ] Pasa pruebas, chequeo de tipos integrado al build y build de producción del módulo.
- [ ] Describe comandos y evidencia reproducible en el PR.

## 12. Orden de integración

El desarrollo es paralelo. El orden siguiente solo controla la integración y las migraciones:

1. Crear `codex/s2-sprint2-integration` desde `main` actualizado.
2. Integrar Lucero y verificar la compatibilidad de recuperación con V1/V2.
3. Integrar Michelle y verificar V3/V4.
4. Integrar Stephano y verificar V5.
5. Integrar José y consolidar router, navegación, OpenAPI y scripts.
6. Ejecutar la matriz completa desde volúmenes vacíos.
7. Abrir PR de integración hacia `main` y exigir CI verde.

Una rama que termina antes puede revisarse y corregirse sin esperar a las demás, pero no se publica parcialmente como Sprint 2 terminado.

## 13. Matriz mínima de pruebas

| Capa | Evidencia requerida |
|---|---|
| Dominio Go | reglas, roles, expiración, semáforo, estados y errores |
| PostgreSQL | Flyway, constraints, RLS, aislamiento y transacciones |
| Concurrencia | primer OWNER, código de orden, idempotencia y tokens de un uso |
| HTTP | sesión, CSRF, rate limit, permisos, códigos y OpenAPI |
| Vue | loading, vacío, error, validación, conflicto y accesibilidad |
| E2E | onboarding, recuperación, orden, equipo, 404 y logout |
| Responsive | 360 px, tableta y escritorio sin scroll horizontal |
| Seguridad | secretos ausentes, tokens hasheados, mensajes anti-enumeración y 404 opaco |

Comandos que deben quedar verdes:

```powershell
docker compose --profile app up --build --wait
docker compose run --rm flyway validate
go -C backend test ./...
go -C backend vet ./...
docker run --rm --volume "${PWD}/backend:/src" --workdir /src golang:1.27-bookworm go test -race ./...
npm --prefix frontend test
npm --prefix frontend run build
npm --prefix frontend run test:e2e
```

Los comandos se ejecutan desde el directorio correspondiente según el script de verificación. Si un script abstrae estos pasos, debe imprimir cada resultado y devolver código distinto de cero al fallar.

## 14. Definición de terminado

- [ ] Una instalación vacía ofrece onboarding en vez de un login sin cuenta.
- [ ] Solo puede reclamarse el primer OWNER una vez, incluso con concurrencia.
- [ ] Login, sesión, cambio y recuperación de contraseña funcionan sin filtrar usuarios.
- [ ] Una ruta inexistente muestra 404; una ruta protegida muestra login.
- [ ] OWNER/ADMIN crea cliente y orden con etapas, semáforo y código único.
- [ ] Reintentos y doble clic no duplican órdenes, códigos, invitaciones o sesiones.
- [ ] OWNER administra miembros y no puede perder el último OWNER activo.
- [ ] Otro taller nunca obtiene datos y recibe 404 neutro cuando corresponde.
- [ ] OpenAPI coincide con handlers y cliente frontend.
- [ ] Flyway migra y valida una base vacía con V1–V5.
- [ ] Todas las pruebas y builds pasan localmente y en CI.
- [ ] La demo completa se repite a 360 px y escritorio desde volúmenes vacíos.
- [ ] No quedan defectos críticos o altos abiertos.
- [ ] Runbook, README y evidencia de demo quedan actualizados.

## 15. Demo final

1. Stephano levanta PostgreSQL, Flyway y la aplicación desde cero.
2. José crea el primer taller y OWNER desde el navegador.
3. Lucero demuestra login, recuperación y página 404.
4. Michelle crea un cliente y una orden, filtra la lista y confirma una etapa.
5. Stephano genera una invitación e incorpora un OPERATOR.
6. El equipo demuestra permisos `403`, aislamiento `404`, logout y restauración de sesión.
7. José ejecuta `verify-sprint2` y muestra CI verde.

El Sprint 2 termina cuando esta demo funciona sin terminal para el usuario final y puede repetirse desde una base vacía.
