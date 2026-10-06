# TallerFlow — Fase 2: órdenes y operación básica

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
