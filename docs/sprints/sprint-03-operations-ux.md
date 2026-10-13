# Operacion visual del taller (Sprint 3)

## Recorrido

1. El propietario o administrador entra a `/app` y ve pedidos activos, urgencias, entregas del dia y carga por estacion.
2. En `/app/orders` puede alternar entre tablero Kanban y lista, buscar por codigo/prenda/cliente, y filtrar por riesgo o fecha de entrega.
3. Cada orden activa aparece en su estacion actual. `Pasar a ...` abre una confirmacion con el codigo, la etapa actual y la siguiente. Cancelar no guarda nada. Al confirmar se consulta el detalle vigente: si la etapa cambio, se actualiza la vista sin moverla; de lo contrario, el backend completa la etapa `IN_PROGRESS` y activa la siguiente. Al completar Entrega, la orden pasa a Completadas.
4. El operario entra a `/app/production`, ve todas las ordenes activas del taller por fecha prometida y abre una para registrar avance o evidencia. Puede buscar por codigo o usar la camara si su navegador admite `BarcodeDetector`; el ingreso manual siempre esta disponible.
5. El numero de orden admite el codigo completo (`OT-000003`), una forma corta (`OT-3`) o solo el numero (`3`) en el tablero y la cola. La busqueda numerica es exacta para evitar confundir ordenes.
6. Si una etapa se completo por error, el propietario o administrador puede usar `Corregir avance` en el detalle de la orden y escribir un motivo. El boton solo aparece en la ultima etapa completada cuando la siguiente sigue en proceso y no tiene fotos. La API vuelve a verificar esas condiciones; la correccion queda registrada en `audit_events`. El operario debe solicitarla a un administrador.

El tablero permite observar las etapas. El avance es una accion explicita para evitar movimientos accidentales y funciona con mouse, teclado y pantalla tactil. La vista de lista conserva una opcion compacta para localizar pedidos rapidamente.

Los filtros de riesgo y `Entrega hoy` son excluyentes. `Todas` restablece el filtro temporal; la busqueda de texto permanece hasta limpiarla. Un resultado vacio ofrece `Ver todas las ordenes`. No se usa arrastrar tarjetas para cambiar datos: el gesto seria menos preciso en pantallas tactiles y ocultaria una accion que debe confirmarse en el servidor.

En el detalle, el estado de entrega aparece en una linea compacta junto a la fecha prometida. El formulario de alta muestra las siete estaciones en orden sin duplicar numeros. Los titulos de paginas internas comparten una escala definida en `frontend/src/styles/tokens.css`; los iconos provienen de Lucide.

## Contratos existentes

- `GET /api/v1/dashboard`: totales operativos del taller.
- `GET /api/v1/orders`: tarjetas del tablero con `current_stage` y semaforo.
- `GET /api/v1/orders/:id`: detalle y posiciones de etapas antes de registrar un avance.
- `PATCH /api/v1/orders/:id/stages/:position`: completa la etapa activa con CSRF; la siguiente transicion ocurre en el servidor.
- `POST /api/v1/orders/:id/stages/:position/reopen`: requiere rol `OWNER` o `ADMIN`, CSRF y `{ "reason": "..." }` de 5 a 300 caracteres. Revierte solo el ultimo avance, bloquea la correccion si la siguiente estacion ya tiene evidencia y escribe `ORDER_STAGE_REOPENED` en auditoria. Responde `409` cuando el avance ya no es corregible.

Clientes usa la API existente para crear y editar nombre, telefono y correo. No se cambio el esquema de la base: esos datos ya existian. Los colores semanticos compartidos estan en `frontend/src/styles/tokens.css`: verde petroleo para marca/acciones, verde para exito, ambar para riesgo, rojo para atrasos o acciones destructivas y azul para informacion.

El flujo actual del backend es Corte, Costura, Acabado, Control de calidad, Planchado, Empaque y Entrega. Hoy la cola es compartida: no hay asignacion individual de ordenes a operarios. El tablero usa esas mismas etapas; si el taller requiere flujos distintos, hay que extender primero el modelo de ordenes y sus reglas de transicion.

## Verificacion

Con la aplicacion iniciada en `http://localhost:8080`:

```powershell
cd frontend
npm run build
npm test
npx playwright test e2e/operations-ui.spec.ts
```

En un equipo que ya tenga Chrome pero no el Chromium de Playwright:

```powershell
$env:E2E_BROWSER_CHANNEL = 'chrome'
npx playwright test e2e/operations-ui.spec.ts
```

La prueba de navegador usa respuestas controladas para verificar la disposicion movil y de escritorio, la busqueda/filtros, la cancelacion de la confirmacion y el movimiento de una tarjeta. No modifica los pedidos de la base local. Para una comprobacion manual con datos reales, entra con una cuenta existente, crea una orden, confirma el avance desde su tarjeta o su detalle y usa `Corregir avance` antes de registrar fotos en la siguiente etapa. Se corrigio tambien el tipo de los parametros JSON del INSERT de auditoria, que PostgreSQL rechazaba y hacia fallar la correccion.

## Siguientes decisiones

- La cola del operario muestra ordenes del taller, no asignaciones personales. Si se necesita un buzon individual, primero hay que definir responsable, reasignacion, permisos y auditoria en el modelo de datos.
- La ficha viajera imprime un codigo legible. La camara solo funciona donde el navegador admite `BarcodeDetector`; antes de prometer un flujo de escaneo universal conviene generar un codigo de barras o QR estandar y probarlo en los dispositivos del taller.
- No se añadieron datos nuevos a clientes u ordenes sin un caso de uso confirmado. El siguiente candidato util seria una referencia externa del pedido, pero requiere decidir su formato y si debe ser unica por taller.
- La correccion de etapas guarda el motivo internamente. Una vista de historial de cambios visible para encargados puede construirse sobre `audit_events` cuando se defina que eventos deben mostrarse y durante cuanto tiempo.
