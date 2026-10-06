<!-- /autoplan restore point: "C:\\Users\\JOSE\\AppData\\Local\\Temp\\tallerflow-orders-autoplan-restore-20260922.md" -->
## Implementation plan
# TallerFlow — Fase 2: órdenes y operación básica, primera versión productiva

## Resultado

Un OWNER o ADMIN crea un cliente y una orden completa desde escritorio o móvil,
consulta sus órdenes filtradas por estado o entrega, y abre el detalle. Un
OPERATOR no puede crear ni editar órdenes. Una solicitud para un recurso de otro
taller responde `404`.

## Alcance

- Clientes activos: crear, listar, consultar y actualizar sin eliminación física.
- Órdenes: crear, listar paginado, consultar detalle y editar datos permitidos
  mientras su estado sea `ACTIVE`.
- Código secuencial único por taller, creado bajo transacción y bloqueo de fila.
- Plantilla de siete etapas al crear una orden. `Pedido confirmado` inicia
  `COMPLETED`; la primera etapa productiva aplicable inicia `IN_PROGRESS`; las
  no aplicables se guardan `SKIPPED`.
- Estados de orden: `ACTIVE`, `PAUSED`, `COMPLETED` y `CANCELLED`.
- Semáforo de entrega: `LATE`, `AT_RISK`, `ON_TIME` y `COMPLETED`.
- Dashboard inicial con conteos por semáforo y órdenes recientes.
- Vistas responsivas: lista de órdenes, formulario guiado de creación y detalle.

## Reglas y seguridad

- Toda lectura y escritura usa el taller del principal autenticado y PostgreSQL
  RLS. Un UUID ajeno no revela su existencia.
- Solo OWNER y ADMIN crean o editan clientes y órdenes. OPERATOR puede consultar
  únicamente lo que su rol permita en la siguiente fase, pero no crea órdenes.
- Producto no vacío, cantidad positiva y fechas coherentes son obligatorios.
- Las fechas comprometidas son `DATE` interpretadas en `America/Lima`; los
  timestamps se guardan en UTC.
- OpenAPI es el contrato de transporte y Flyway la única fuente del esquema.

## Fuera de alcance

- Registrar avances, correcciones, problemas, pausa/reanudación/cancelación,
  responsables, adjuntos y tracking externo: Fase 3 o posterior.
- Pagos, notificaciones, importaciones masivas, eliminación física y multi-taller
  por usuario.

## Dependencias

- Fase 1 integrada: sesión propia, principal de taller, roles, RLS, Flyway,
  Vue con guard de sesión, Compose y CI.
- La migración de clientes debe preceder a la migración de órdenes y etapas.

## Criterio de aceptación

- OWNER/ADMIN crea cliente y orden con siete etapas correctas y código único.
- La lista pagina, filtra y muestra semáforo correcto a 360 px y escritorio.
- OPERATOR obtiene `403` al crear o editar; otro taller obtiene `404`.
- Una repetición de creación no duplica una orden ni un código.
- Pruebas Go, migraciones Flyway, Vue y E2E cubren creación, aislamiento,
  numeración concurrente, validaciones y experiencia móvil.
## Revisión productiva: alcance y precedencia

Este módulo deja de planificarse como un MVP desechable. Por decisión de producto del 2026-09-22, esta es una primera versión productiva: alcance funcional pequeño, pero datos, seguridad, recuperación y operación aptos para uso real. Esta precedencia es local al módulo de órdenes; se conserva el stack Go + Gin + GORM, PostgreSQL con Flyway, Vue 3 + TypeScript + Vite, Docker y autenticación propia.

Entran ahora: clientes, creación/listado/detalle/edición de órdenes, código secuencial, etapas como instantánea, semáforo, dashboard, OpenAPI, auditoría mínima, observabilidad, pruebas reales y puerta de backup/restauración. Quedan preparados pero no expuestos: asignaciones, avances append-only, estados operativos, plantillas configurables, tracking, evidencias, S3, réplica y HA.

## Puerta de inicio

Antes de escribir código se comprobará que Fase 1 está integrada en `main`: sesión propia, CSRF, principal con `workshop_id` y rol, RLS mediante `SET LOCAL`, rol de aplicación sin `BYPASSRLS`, request ID, Compose y Flyway desde base vacía. Solo después se numeran migraciones; nunca se reutiliza ni edita una aplicada y `AutoMigrate` permanece prohibido.

## Decisiones de datos y concurrencia

- `clients`: UUID, taller, datos de contacto, estado activo/inactivo y timestamps; no hay borrado físico. Índice de lista `(workshop_id, status, name, id)`; teléfono y email opcionales no se fuerzan únicos.
- `workshop_counters`: una fila única por taller, bloqueada al reservar el siguiente código.
- `orders`: unique `(workshop_id, code)`, cliente del mismo taller, cantidad positiva, fechas coherentes, estado, versión/ETag, creador y timestamps; índices para listado por taller, estado, entrega e ID.
- `order_stages`: snapshot con claves `order_confirmed`, `cutting`, `sewing`, `printing_or_embroidery`, `finishing`, `quality_control` y `delivery`; unique por orden/secuencia y orden/clave. `order_confirmed` nace completada; la primera aplicable inicia en progreso; las no aplicables quedan omitidas.
- Idempotencia persistida: unique `(workshop_id, actor_id, operation, key)`, hash de payload y respuesta. Misma clave con payload distinto devuelve `409`.
- `audit_events` append-only: actor, taller, recurso, acción, resultado y request ID; sin cookies, CSRF, tokens, email completo ni cuerpo de petición.

El semáforo se calcula, no se edita: `COMPLETED`; `LATE` si venció y sigue activa; `AT_RISK` si vence hoy/mañana y no está en la última etapa; `ON_TIME` en otro caso. “Hoy” usa la zona del taller. Las etapas no se reordenan ni reactivan desde edición: hacerlo requiere el historial de Fase 3.

## Edición, permisos y API

La versión crea órdenes `ACTIVE`; reserva los demás estados sin exponer transiciones hasta Fase 3. `PATCH` solo cambia cliente, producto, descripción, cantidad y fechas de una orden activa. Exige `If-Match`; una versión antigua devuelve `409 ORDER_VERSION_CONFLICT` y la UI conserva el borrador. `POST /orders` exige `Idempotency-Key`; contador, orden, etapas, idempotencia y auditoría se confirman en una única transacción.

OWNER y ADMIN pueden crear, listar, leer y editar clientes, órdenes y dashboard. OPERATOR no recibe lectura temporalmente amplia: sin asignaciones no existe un filtro seguro; Fase 3 habilitará solo órdenes asignadas. Otro taller y un UUID inexistente responden igual, `404`. El caso de uso autoriza, el repositorio usa contexto de taller transaccional y RLS es la segunda barrera.

OpenAPI documentará: `GET/POST /clients`, `GET/PATCH /clients/{id}`, `GET/POST /orders`, `GET/PATCH /orders/{id}`, `GET /orders/{id}/stages` y `GET /dashboard/summary`. Listas usan cursor opaco, límite acotado, búsqueda, filtros validados y orden estable. Detalle devuelve ETag y etapas; no devuelve capacidades futuras. Éxitos usan `data/meta`; errores incluyen código, mensaje, detalles seguros y request ID. Se especifican 400, 401, 403, 404 opaco, 409, 422, 429 y 500.

## Entregables por fases

| Entrega | Resultado verificable |
|---|---|
| 0. Base real y contrato | Fase 1 comprobada, esquema inventariado, OpenAPI/DTOs acordados; sin crear migración aún |
| 1. Clientes seguros | casos de uso, RLS, API, UI y auditoría aislados por taller |
| 2. Órdenes transaccionales | contador, snapshot, constraints, idempotencia y creación concurrente sin duplicados |
| 3. Consulta y edición | cursor, filtros, dashboard, semáforo, detalle y conflicto de versión sin N+1 |
| 4. UX y contrato integrado | responsive, accesibilidad, estados y E2E contra PostgreSQL real |
| 5. Release productivo | métricas, runbook, health checks, backup y restauración aislada demostrada |

## Calidad, seguridad, recuperación y riesgos

Pruebas: dominio (reglas, fechas Lima, semáforo, roles), PostgreSQL real (Flyway, constraints, RLS con rol de aplicación), concurrencia (creaciones paralelas, reenvío de clave, payload distinto y ETag obsoleto), HTTP (sesión/CSRF/origen/errores/OpenAPI), Vue (loading/vacío/error/conflicto/accesibilidad) y E2E (OWNER/ADMIN, OPERATOR `403`, otro taller `404`, móvil 360 px y escritorio). Release exige `go test -race`, `go vet`, frontend test/lint/build, `vue-tsc`, `flyway validate`, Compose sano y cero defectos críticos/altos.

Logs JSON incluyen request ID, ruta, resultado y latencia sin PII completa ni secretos. Métricas: latencia/error, conflictos de idempotencia/versión, denegaciones y tamaño de listas. Alertas: 5xx, ready no saludable, migración o backup fallido. Docker usa imágenes fijadas, secretos externos, TLS en Caddy y PostgreSQL privado. Migraciones forward-only deben permitir rollback de aplicación sin revertir SQL; el runbook define desactivación o migración correctiva.

El release se bloquea si Fase 1 es solo mock, si hay fuga entre talleres, código duplicado, pérdida silenciosa de edición, N+1 no medido, auditoría/logs con PII o backup sin restauración aislada. La restauración debe verificar integridad de clientes/órdenes, tiempo de recuperación, responsable, frecuencia, retención, destino cifrado y alerta de fallo.

## Decisiones de producto validadas por la revisión estratégica

### Trazabilidad mínima que sí entra

Para que las etapas no sean decorativas, la primera versión productiva incluye una única transición administrativa y auditada: OWNER o ADMIN puede confirmar que la etapa actual aplicable terminó y avanzar a la siguiente aplicable en una transacción. No incluye porcentajes, avances incrementales, correcciones, incidencias, asignaciones ni pausas: esas capacidades permanecen en Fase 3. La UI identifica esta capacidad como “confirmar etapa”, no como registro completo de producción.

La transición exige orden `ACTIVE`, versión/ETag actual y una única etapa activa. Registra actor, timestamps, etapas origen/destino y request ID en auditoría. Al terminar `delivery`, no completa ni cancela automáticamente la orden en esta primera versión: el estado operativo completo se incorpora con el historial append-only de Fase 3. Si una transición falla, no cambia ninguna etapa. Se añade `POST /orders/{id}/stages/{stageId}/advance` con `If-Match` y respuesta `409` ante versión o etapa obsoleta.

### Datos operativos mínimos y política de códigos

Además de cliente, producto, cantidad y fechas, la creación define unidad de medida, especificación de producto y notas internas con límites explícitos. Prioridad queda como clasificación controlada (`NORMAL`, `HIGH`) solo si la entrevista de descubrimiento confirma que cambia la decisión diaria; de lo contrario se difiere para no inventar complejidad. No se modelan cotizaciones, moneda, recepción ni variantes hasta que un caso de uso validado lo exija.

El código es una referencia de negocio estable y no reutilizable, único por taller, con formato versionado acordado antes de implementación. Puede contener huecos tras una transacción abortada o una reserva que no llega a confirmarse; esos huecos no representan una orden perdida. La misma clave de idempotencia devuelve siempre el mismo código y orden. Una referencia pública futura se modelará aparte del UUID interno.

### Semáforo, dashboard y validación de adopción

El semáforo excluye `CANCELLED`; `PAUSED` se reporta como pausada, no como `ON_TIME`; y una fecha de entrega ausente se rechaza en esta versión. La fórmula se prueba con el reloj inyectado y la zona `America/Lima`. El dashboard de Fase 2 sirve a OWNER/ADMIN para priorizar órdenes atrasadas o en riesgo, no para medir rendimiento ni ofrecer una cola de operador aún inexistente.

Antes de cerrar alcance, se valida con talleres: tiempo actual de alta, principal causa de retraso, decisión diaria de dueño/admin y campos que no pueden faltar. Métricas de éxito: mediana de tiempo para crear orden, porcentaje con fecha y especificación completa, uso del dashboard para abrir una orden prioritaria y discrepancias detectadas entre la etapa registrada y la operación real. Estos datos deciden la siguiente fase, no se usan para prometer una métrica sin línea base.

## Contrato de UX por vista y estado

### Jerarquía y responsive

Dashboard: primero contadores accionables por semáforo, después órdenes atrasadas/en riesgo y recientes; tocar un contador filtra la lista. Lista: título, CTA `Nueva orden` para rol autorizado, filtros activos visibles y cada orden con código, cliente, producto, fecha, etapa actual y semáforo. Detalle: cabecera con código, semáforo textual, cliente, entrega y editar; luego etapa actual y timeline; finalmente especificación, cantidad, unidad y notas. Formulario: cliente, producto/cantidad/unidad, proceso aplicable y fechas, con notas al final.

Bajo 768 px la lista se convierte en tarjetas de una columna, sin scroll horizontal: código, semáforo y fecha arriba; cliente, producto y etapa en cuerpo. Los filtros se abren en panel o diálogo con contador, `Aplicar` y `Limpiar`; en escritorio se muestran en línea. Búsqueda, filtros y cursor se reflejan en URL; cambiar un filtro reinicia cursor y volver desde detalle conserva consulta y posición.

### Formularios, etapas y conflictos

El cliente se busca dentro del formulario y ofrece CTA `Crear cliente` sin abandonar el flujo. Validación por campo al perder foco, resumen de errores al enviar, errores `422` junto al control, límites visibles para especificación/notas y confirmación al abandonar un borrador modificado. La creación exige al menos una etapa productiva aplicable; muestra un resumen de proceso antes de guardar. El snapshot no se edita después.

Tras crear, se navega al detalle y se anuncia `Orden {código} creada`, con CTA para crear otra. El detalle muestra estados de etapa como lista con icono y texto: completada, actual, pendiente y no aplicable. Solo la etapa actual presenta `Confirmar etapa` para OWNER/ADMIN, con efecto explicado, confirmación breve, botón ocupado y timeline/ETag actualizados. Los avances incrementales siguen fuera de alcance.

Un conflicto ETag abre un diálogo accesible con la versión modificada por otra persona, campos locales y momento del cambio. Ofrece `Recargar y descartar mi borrador` y `Revisar cambios`; solo permite nuevo envío con ETag fresco y confirmación de campos a conservar. No hay doble envío.

### Matriz de estados y permisos UI

| Situación | Conducta definida |
|---|---|
| Cuenta sin órdenes | explicación y CTA crear para OWNER/ADMIN |
| Filtros sin coincidencias | conserva filtros y ofrece `Limpiar filtros` |
| Carga | skeleton con estructura final; página siguiente indica carga al final sin cubrir resultados |
| Error recuperable | explica qué ocurrió, causa probable y siguiente acción; request ID copiable y `Reintentar` |
| Dashboard parcial | conserva bloques correctos, marca el bloque fallido y permite reintento localizado |
| Orden inexistente o ajena | 404 neutro sin datos ni nombre de taller |
| OPERATOR | no ve CTAs ni navegación de clientes/órdenes; URL manual recibe 403 explícito sin datos |

Todos los controles móviles miden al menos 44×44 px, tienen etiqueta visible y contraste WCAG AA. Errores usan `aria-describedby`; el primer error recibe foco al enviar; paneles, confirmaciones y conflictos contienen y restauran foco; teclado cubre todo el flujo; `aria-live` anuncia creación, guardado, error y confirmación de etapa. Semáforos siempre combinan etiqueta, icono y fecha con “hoy/mañana”, incluido `PAUSED`; una ayuda breve explica sus umbrales.

Los E2E cubren crear cliente desde una orden, volver de detalle con filtros conservados, conflicto sin pérdida de borrador, confirmación de etapa correcta/obsoleta, 404 opaco, estados vacío/error/parcial y navegación por teclado a 360 px.

## Contrato para desarrollo y operación interna

OpenAPI versionado en `/api/v1` es fuente única: esquemas reutilizables de petición, respuesta, error, cursor, ETag e idempotencia; ejemplos completos por operación y cabecera; tipos y cliente TypeScript generados para Vue. CI valida/linta el documento, verifica compatibilidad y evita divergencia con handlers. Los cambios son aditivos dentro de v1; deprecación y retiro tienen ventana documentada. Cursor es opaco, con límite por defecto/máximo, campos de orden permitidos y filtros con semántica estable.

El catálogo versionado de errores define para cada código: significado, campos `details` permitidos, si se puede reintentar, acción de usuario/operador y ayuda/runbook. El frontend decide por código, no por texto. Se publican ejemplos para 400, 401, 403, 404, 409, 422, 429 y 500, sin convertir el 404 opaco en una fuga de información.

La entrega 0 añade onboarding reproducible: prerrequisitos, `.env.example` sin secretos, comandos para Compose, Flyway validate/migrate, usuario+taller local controlado, sesión/CSRF de prueba y ejemplo HTTP que crea cliente y orden. Se mide TTHW de un desarrollador nuevo y se apunta a menos de diez minutos. No se construye CLI ni portal de API como condición de esta fase.

Las migraciones siguen expand/contract: primero cambio aditivo compatible, luego aplicación tolerante a esquema previo/nuevo, verificación, y solo en entrega posterior retiro de compatibilidad. El runbook incluye prechecks, lock/timeout de Flyway, evaluación de índices costosos, smoke test, pausa y migración correctiva. Request/trace ID se propaga de Gin a auditoría y acceso a datos; métricas tienen etiquetas de operación/error con cardinalidad acotada. Un runbook de diagnóstico define consultas permitidas para Flyway/RLS y redacción de PII.

## Review record

<!-- autoplan-accepted:ceo -->
- La primera entrega de órdenes se trata como versión productiva acotada: no amplía a microservicios, BaaS ni alta disponibilidad automática.
- Incluye clientes y órdenes con idempotencia, control de versión, aislamiento, auditoría mínima, observabilidad y recuperación verificable.
- OPERATOR no recibe acceso a órdenes hasta que la asignación segura exista en Fase 3.
<!-- /autoplan-accepted:ceo -->

<!-- autoplan-accepted:design -->
- Clientes y órdenes especifican creación, lista, detalle y edición responsive con estados de carga, vacío, error y conflicto; semáforo textual, teclado, foco y controles de 44 px.
<!-- /autoplan-accepted:design -->

<!-- autoplan-accepted:dx -->
None: TallerFlow no es una herramienta para desarrolladores y esta fase no crea una API pública ni CLI de autoservicio.
<!-- /autoplan-accepted:dx -->

<!-- autoplan-accepted:eng -->
- La implementación exige Flyway forward-only, transacciones, RLS con rol de aplicación, constraints, ETag, idempotencia, pruebas de concurrencia y evidencia de restore antes de release.
<!-- /autoplan-accepted:eng -->

<!-- AUTONOMOUS DECISION LOG -->
## Decision Audit Trail

| # | Phase | Decision | Classification | Principle | Rationale | Rejected |
|---:|---|---|---|---|---|---|
| 1 | CEO | Planificar una primera versión productiva, no un MVP desechable | User direction | Completeness | Dirección explícita de producto, manteniendo el alcance funcional acotado | Mantener el plan MVP literal o reescribir todo el producto |
| 2 | CEO | No dar lectura global al OPERATOR sin asignaciones | Mechanical | Explicit over clever | Un permiso temporal amplio expondría datos del taller | Visibilidad por rol sin relación orden-operario |
| 3 | Eng | Persistir idempotencia y usar ETag para mutaciones | Mechanical | Completeness | Cubre reintentos de red y edición concurrente sin duplicar ni sobrescribir | Last-write-wins o reintentos ciegos |
| 4 | Eng | Mantener etapas como instantánea con claves estables | Mechanical | Explicit over clever | Conserva trazabilidad y permite plantillas futuras sin cambiar órdenes históricas | Aplicar una plantilla mutable retroactivamente |

## GSTACK REVIEW REPORT

### Resultado

El plan de clientes y órdenes fue revisado frente a la especificación y la hoja de ruta. La corrección de producto se registró como precedencia local: alcance funcional contenido, pero con requisitos explícitos de datos, seguridad, operación, recuperación y evolución.

| Revisión | Estado | Hallazgos aceptados |
|---|---|---:|
| CEO/alcance | Revisado | 2 |
| Diseño/UI | Revisado | 1 |
| DX | Omitido con motivo | 0 |
| Ingeniería | Revisado | 2 |

### Cobertura y riesgo abierto

La única dependencia bloqueante es verificar que Fase 1 esté realmente integrada en `main`. El árbol principal contiene documentación y worktrees; antes de crear migraciones o código, la entrega 0 debe demostrar sesión, CSRF, roles, RLS y Flyway en PostgreSQL real.

### Veredicto

Plan listo para aprobación de alcance. Ejecutar solo después de la puerta de Fase 1 y de comprometer pruebas reales y restore aislado como condiciones de release.

NO UNRESOLVED DECISIONS
