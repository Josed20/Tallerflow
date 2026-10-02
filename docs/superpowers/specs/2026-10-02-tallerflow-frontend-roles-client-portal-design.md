# TallerFlow — Frontend por roles y portal de cliente

- **Fecha:** 2026-10-02
- **Estado:** diseño aprobado para planificación
- **Referencia visual:** proyecto Stitch `TallerFlow MVP Design System`
- **Objetivo:** cada persona ve únicamente las acciones y el estado operativo que le corresponden, y el cliente consulta de forma privada el avance de todos sus pedidos.

## 1. Decisiones de producto

TallerFlow tiene dos espacios de acceso independientes:

1. **Espacio interno**, para `OWNER`, `ADMIN` y `OPERATOR`, autenticado por la sesión actual del producto.
2. **Portal de cliente**, de solo lectura, autenticado con un enlace seguro enviado al correo del cliente. No reutiliza ni expone cuentas internas del taller.

El enlace crea o restaura una sesión de cliente limitada al par `(workshop_id, client_id)`. Desde el portal el cliente ve todos sus pedidos de ese taller. No puede crear, editar, cancelar, adelantar etapas ni consultar datos de otros clientes.

## 2. Sistema visual

La implementación toma como guía Stitch, no como código fuente:

- Marca consistente: `TallerFlow`.
- Tipografía: Manrope.
- Primario: `#0D9488`; fondo oscuro de navegación: `#0F172A`; azul de apoyo: `#0284C7`; texto neutral: `#64748B`.
- Interfaces sobrias, con tarjetas, tablas responsive, filtros visibles y estados semánticos claros.
- Todo control interactivo conserva foco visible, etiqueta accesible y objetivo táctil mínimo de 44 px.

No se incorporan conceptos de inventario, turnos, AQL, mermas, textiles o ERP que no formen parte del flujo de órdenes.

## 3. Matriz de navegación y permisos

| Ruta | OWNER | ADMIN | OPERATOR | CLIENT |
|---|---:|---:|---:|---:|
| `/app` panel interno | sí | sí | sí, producción | no |
| `/app/clients` | crear, editar y listar | crear, editar y listar | no | no |
| `/app/orders` | crear, listar y filtrar | crear, listar y filtrar | listar y filtrar | no |
| `/app/orders/new` | sí | sí | no | no |
| `/app/orders/:id` | consultar y acciones administrativas permitidas | consultar y acciones administrativas permitidas | avanzar solamente la etapa actual | no |
| `/app/team` | administrar equipo | administrar según permisos existentes | no | no |
| `/portal/access` | no | no | no | solicitar enlace por correo |
| `/portal/orders` | no | no | no | listar sus pedidos |
| `/portal/orders/:id` | no | no | no | detalle y línea de tiempo propios |

El guard del router mejora la experiencia, pero nunca se considera la barrera de seguridad. El backend debe responder `403` o `404` de forma neutra según el contrato de cada recurso.

## 4. Pantallas internas

### Panel de OWNER y ADMIN

El dashboard presenta métricas de órdenes activas, a tiempo, en riesgo, atrasadas y terminadas, junto con órdenes recientes. Los accesos principales son **Clientes**, **Órdenes**, **Nueva orden** y **Equipo**. Las tarjetas de estado navegan al listado filtrado correspondiente.

### Panel de OPERATOR

El operador recibe una cola de producción compacta: órdenes activas, etapa actual, fecha prometida y semáforo. El detalle muestra la línea de siete etapas y una sola acción principal sobre la etapa en progreso: **Iniciar** o **Completar**. No se muestran acciones de clientes, equipo ni creación de órdenes.

### Clientes

OWNER y ADMIN ven búsqueda, lista, estado, correo y teléfono. Pueden crear y editar sin borrado físico. Los estados loading, vacío, error y formulario inválido se incluyen como pantallas reales.

### Órdenes

El listado conserva búsqueda, filtros por semáforo/estado, tabla o tarjetas responsive, responsable cuando exista, fechas y botón **Nueva orden** para quien tenga permiso. El texto para compartir usa “enlace privado para compartir por WhatsApp”.

El detalle muestra código, cliente, producto, cantidad, fecha prometida, semáforo y etapas. Los estados pausada, cancelada y corrección de avance quedan explícitos en la UI, pero no se habilitan hasta que el backend proporcione transiciones auditadas. Los modales de confirmación son parte del diseño, no deben fingir que la acción se ejecutó.

## 5. Portal del cliente

1. En `/portal/access`, el cliente escribe su correo y solicita un enlace. La respuesta es anti-enumeración: no confirma si el correo existe.
2. El enlace temporal lleva a una ruta de consumo que crea una sesión de portal segura y redirige a `/portal/orders`.
3. El listado muestra todos los pedidos del cliente en el taller, con código, producto, fecha prometida, semáforo y estado.
4. El detalle muestra solamente información compartible: avance por etapas, última actualización y fecha prometida. No expone usuarios internos, notas privadas, credenciales ni datos de otros clientes.
5. La sesión expira y puede cerrarse; el cliente solicita otro enlace para regresar.

## 6. Contratos frontend antes de backend

Se crearán módulos Vue y tipos explícitos para el portal, sin simular persistencia:

- `POST /api/v1/client-portal/access-links` para solicitar acceso.
- `POST /api/v1/client-portal/access-links/consume` para consumir un token.
- `GET /api/v1/client-portal/session` para restaurar sesión de cliente.
- `GET /api/v1/client-portal/orders` y `GET /api/v1/client-portal/orders/{id}` para datos limitados al cliente autenticado.
- `POST /api/v1/client-portal/logout` para cerrar sesión.

Estos contratos quedan como dependencia de la fase backend. Las pantallas frontend deben mostrar un estado útil cuando el endpoint aún no esté disponible, sin inventar datos ni conceder acceso.

## 7. Estados y errores

Cada vista debe cubrir loading, vacío, error recuperable, sin permiso y acción ocupada. Las mutaciones internas envían el CSRF de la sesión actual. Las actualizaciones de etapa requieren la versión/ETag que defina el backend y muestran un conflicto cuando otra persona modificó la orden antes.

## 8. Pruebas y aceptación

- Router: cada rol llega solo a rutas permitidas; el cliente nunca cae en `/app`.
- Vistas: navegación, loading, vacío, error, teclado y permisos visibles.
- API clients: mutaciones internas envían CSRF; portal usa su sesión separada.
- Responsive: 360 px, tableta y escritorio sin scroll horizontal.
- E2E posterior al backend: OWNER crea cliente y orden; OPERATOR completa la siguiente etapa; CLIENT consulta solo sus pedidos mediante enlace seguro.

## 9. Fuera de alcance de esta fase

- Registro de clientes con contraseña.
- Edición de producción, pagos, fotos, adjuntos, WhatsApp automático o notificaciones push.
- Asignación de órdenes a operarios hasta que exista el modelo backend correspondiente.
- Implementar acciones de pausa, cancelación o corrección antes de sus reglas, auditoría y endpoints.
