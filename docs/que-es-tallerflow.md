# ¿Qué es TallerFlow?

> **Cada orden visible, de Corte a Entrega.**

TallerFlow es una plataforma web y móvil para **talleres de confección**. Permite seguir cada pedido desde que entra al taller hasta que se entrega. El dueño ve todas sus órdenes en un tablero, el operario registra el avance desde su celular con fotos como evidencia, y el cliente final sigue su pedido con un link, sin instalar nada ni llamar.

Nace en Gamarra (Lima), donde miles de talleres coordinan su producción por WhatsApp, llamadas, cuadernos y Excel.

---

## 1. El problema

En un taller de confección típico, la información de cada pedido está repartida en mensajes, llamadas, papeles y la memoria de las personas. Eso causa:

- **Falta de visibilidad:** el dueño no sabe qué pedido está atrasado ni en qué estación está cada uno, hasta que pregunta.
- **Retrasos que se descubren tarde:** el atraso se nota el día de la entrega, cuando ya no hay margen.
- **Llamadas constantes:** el cliente (una marca, un mayorista) llama o escribe para preguntar "¿cómo va mi pedido?", y alguien deja de producir para responder.
- **Doble registro y errores:** lo que se anota en el cuaderno no coincide con lo que se dijo por WhatsApp.
- **Sin evidencia:** cuando hay un reclamo, no hay forma de mostrar en qué estado estaba la prenda en cada etapa.

## 2. La solución

TallerFlow reemplaza esa información dispersa por **una sola fuente de verdad compartida**:

| Para quién | Qué le da TallerFlow |
|---|---|
| **Dueño o administrador** | Un panel que dice de un vistazo cuántas órdenes van atrasadas, cuáles salen hoy y cuánta carga tiene cada estación |
| **Operario** | Una pantalla simple en el celular para avanzar la orden a la siguiente estación, subir una foto y escanear la ficha |
| **Cliente final** | Un link privado que muestra en qué estación va su pedido, la fecha prometida y las fotos de avance |

Resultado: menos llamadas, menos reprocesos, retrasos detectados a tiempo y un cliente que confía en el taller.

## 3. Para quién es

| Perfil | Descripción | Rol en el negocio |
|---|---|---|
| **Taller tradicional** | De 5 a 15 operarios; trabaja con WhatsApp, cuaderno y a veces Excel | Cliente pagador potencial. Necesita algo simple, no un ERP |
| **Taller en crecimiento** | De 10 a 30 operarios, con varias órdenes en paralelo | **Cliente objetivo inicial.** Sufre el desorden todos los días y está más dispuesto a pagar por control |
| **Marca o empresa que terceriza** | Marca de moda, mayorista, retail o cliente corporativo que manda a confeccionar | No paga al inicio, pero exige transparencia e influye en que el taller adopte TallerFlow |

## 4. Cómo funciona: la vida de una orden

```text
 1. Recepción      El dueño registra la orden: cliente, prenda, cantidad, fecha prometida.
        │          TallerFlow le asigna un código (OT-000003).
        ▼
 2. Ficha viajera  Se imprime la ficha con el código y viaja con las prendas por el taller.
        │
        ▼
 3. Producción     La orden pasa por 7 estaciones:
        │          Corte → Costura → Acabado → Control de calidad → Planchado → Empaque → Entrega
        │          En cada una, el operario escanea o busca la ficha, toca "Pasar a…"
        │          y sube una foto como evidencia.
        ▼
 4. Seguimiento    El cliente abre su link y ve: "Tu pedido está en Costura", 3 de 7 estaciones,
        │          entrega prometida el viernes, y las fotos del avance.
        ▼
 5. Entrega        Al completar Entrega, la orden pasa a Completadas.
```

**Semáforo de entrega.** Cada orden activa tiene un estado según su fecha prometida y su avance:

| Estado | Significado | Cómo se ve |
|---|---|---|
| **A tiempo** | Va bien para la fecha prometida | Verde, con ● |
| **En riesgo** | Puede no llegar si no se acelera | Ámbar, con ▲ |
| **Atrasada** | Ya pasó o no llegará a la fecha | Rojo, con ■! (es el único rojo sólido de la app) |
| **Completada** / **Cancelada** | Terminó | Grafito, con ✓ o ✕ |

El estado nunca depende solo del color: siempre lleva una palabra y una forma, para quien no distingue colores.

## 5. Qué hace hoy (MVP funcionando)

**Cuentas y equipo:**

- Inicio de sesión seguro, cierre de sesión y cambio obligatorio de contraseña en el primer ingreso.
- Recuperación de contraseña por correo.
- Invitación de miembros al equipo con un link y un rol asignado. El dueño puede desactivar o cambiar el rol de cada miembro.

**Operación del taller:**

- **Panel:** cuántas órdenes van atrasadas (o "Todo a tiempo"), las que necesitan atención, las que salen hoy y la carga por estación.
- **Órdenes:**
  - vista de tablero por estaciones (tipo Kanban) o de lista;
  - búsqueda por código, prenda o cliente;
  - filtros por riesgo y por "Entrega hoy".
- **Avance por estación:** con un diálogo de confirmación, que funciona con mouse, teclado y pantalla táctil.
- **Corregir avance:** el dueño o administrador puede revertir el último avance con un motivo, que queda registrado en la auditoría.
- **Fotos de evidencia** en cada estación.
- **Escáner:** abre una orden con la cámara (códigos QR o de barras) o escribiendo el código: `OT-000003`, `OT-3` o solo `3`.
- **Ficha viajera** imprimible para cada orden.
- **Clientes:** alta y edición de nombre, teléfono y correo.

**Para el cliente final:**

- **Link de seguimiento** por orden. Muestra la estación actual, la costura de avance (por ejemplo, 3 de 7), la fecha prometida, el estado y las fotos. Hay un link activo por orden y generar uno nuevo invalida el anterior.

**Calidad:**

- 81 pruebas unitarias, pruebas de navegador (E2E) en un celular de 360 px y pruebas de seguridad de la base de datos.

## 6. Quién puede hacer qué

| Nivel | Quién | Qué ve y hace |
|---|---|---|
| **Propietario (OWNER)** | El dueño del taller | Todo: panel, órdenes, clientes, equipo y corrección de avances |
| **Administrador (ADMIN)** | Quien el dueño designe | Lo mismo que el propietario |
| **Operario (OPERATOR)** | Personal de producción | Solo **Órdenes**: avanzar estaciones, subir fotos y escanear la ficha |
| **Cliente final** | El cliente del taller, sin cuenta | Una sola orden, mediante su link privado |
| **Plataforma** *(Sprint 3)* | Los dueños de TallerFlow | Lista de talleres y métricas; suspender, reactivar y eliminar talleres. **No** ve órdenes ni clientes |

## 7. Por qué es seguro compartir un mismo sistema entre muchos talleres

TallerFlow es **multi-tenant**: un solo sistema atiende a muchos talleres, y cada taller ve **solo sus datos**.

- **Aislamiento en la base de datos:** usa *Row-Level Security* de PostgreSQL. Aunque hubiera un error en el código, la base no devuelve datos de otro taller.
- **Contraseñas:** se guardan con Argon2id, el algoritmo recomendado hoy. Nunca se guardan en texto.
- **Sesiones:** son opacas y revocables, con cookie protegida, protección CSRF y límite de intentos contra ataques de fuerza bruta.
- **Links de seguimiento:** usan tokens aleatorios que solo se guardan cifrados (hash). Sin el link exacto, no se puede ver la orden.
- **Auditoría:** las acciones sensibles, como corregir un avance o cambiar un rol, quedan registradas.
- **Privacidad:** el equipo de TallerFlow no mira el contenido de las órdenes de los talleres, y la consola de plataforma está diseñada para no poder hacerlo.

## 8. Diseño: "Ficha viajera"

La identidad visual se inspira en la **ficha de papel que viaja con las prendas** por el taller. Está definida en `DESIGN.md`.

- **Logo:** una etiqueta colgante con 7 puntadas, una por estación.
- **Color de marca:** azul denim (`#1F3A68`). El rojo se reserva solo para "Atrasada", así el que aparece rojo siempre es urgente.
- **Órdenes como tickets** con una esquina recortada, y una **costura de 7 puntadas** que se va cosiendo a medida que la orden avanza.
- **Tipografías:** Archivo para los títulos y Atkinson Hyperlegible para el texto, elegida por ser muy legible en celulares de gama baja y a pleno sol en el taller.
- **Pensada primero para celular:**
  - botones de al menos 44 px para usar con los dedos, incluso con guantes o manos ocupadas;
  - todo se puede usar con teclado;
  - contraste suficiente para leer bajo la luz del taller.

## 9. Cómo está construido

| Capa | Tecnología |
|---|---|
| Frontend web | Vue 3 + TypeScript, Vite, Pinia, TanStack Query, Tailwind CSS |
| App móvil *(Sprint 3)* | El mismo código Vue empaquetado con Capacitor para Android |
| Backend (API) | Go con Gin y GORM |
| Base de datos | PostgreSQL 18 con Row-Level Security y migraciones con Flyway |
| Servidor web | Caddy (HTTPS y un solo dominio para web y API) + nginx |
| Infraestructura | Docker Compose; en producción, un VPS con Cloudflare delante |
| Pruebas | Go test, Vitest, Playwright (E2E) y pruebas SQL de seguridad |
| Integración continua | GitHub Actions |

```text
  Navegador / App Android ──▶ Cloudflare ──▶ VPS: Caddy ──▶ API Go ──▶ PostgreSQL (RLS)
                                                   └──▶ nginx (web Vue)
```

La web y la app usan **la misma API y la misma base de datos**, así que un cambio hecho en el celular se ve al instante en la computadora. Ni la web ni la app se conectan directo a la base: solo la API entra a ella.

## 10. Modelo de negocio (propuesto)

TallerFlow se ofrece como **SaaS**: el taller paga una suscripción mensual y no instala nada.

- **Lo que paga el taller:** visibilidad, menos llamadas y un cliente más satisfecho.
- **Por qué funciona como SaaS:** un solo sistema para todos los talleres reduce el costo por taller. El costo de infraestructura del piloto es de unos **S/ 85 a S/ 100 al mes** en total.
- **Planes y cobro** (prueba gratis y luego pago mensual) están planificados para el Sprint 4. El precio se definirá con lo que se aprenda en el piloto.
- **Crecimiento:** cada link de seguimiento que el taller comparte muestra la marca TallerFlow a sus clientes, que a su vez trabajan con otros talleres.

## 11. Estado y hoja de ruta

| Etapa | Contenido | Estado |
|---|---|---|
| **Sprint 1** | Cimientos seguros: base de datos con RLS, autenticación, roles, Docker y CI | ✅ Terminado |
| **Sprint 2** | Clientes, órdenes, 7 estaciones, fotos, escáner, equipo, recuperación de contraseña, link del cliente y rediseño "Ficha viajera" | ✅ Terminado |
| **Sprint 3** (19 al 30 de octubre de 2026) | Ver el detalle más abajo | 🔄 En curso |
| **Sprint 4** | Ver el detalle más abajo | 📋 Planificado |

**Sprint 3: listo para operar como SaaS, en web y celular.**

- Registro abierto de talleres con verificación de correo.
- Invitaciones por correo.
- App Android.
- Deshacer un avance hecho por error.
- Consola de plataforma, con suspensión y eliminación de talleres.
- Dominio propio con HTTPS, respaldos diarios y monitoreo.

**Sprint 4.**

- Planes y cobro.
- Editar órdenes.
- Fotos en almacenamiento externo.
- App iOS.
- Exportar los datos del taller.
- Código QR impreso en la ficha viajera: hoy la ficha muestra el código en texto y el escáner ya lee QR.

El detalle del Sprint 3 está en `docs/sprints/sprint-03-team-plan.md`.

## 12. Equipo

| Persona | Área |
|---|---|
| **José** | Líder técnico: autenticación, registro de talleres, despliegue e integración |
| **Lucero** | Frontend, experiencia de usuario, correos y marca |
| **Michael** | Multi-tenancy, talleres y roles, consola de plataforma y órdenes |
| **Stephano** | Base de datos, infraestructura, CI/CD y app móvil |

Proyecto de startup del Ciclo IX.

## 13. Glosario

| Término | Significado |
|---|---|
| **Orden (OT)** | Un pedido de confección: cliente, prenda, cantidad y fecha prometida. Se identifica con un código como `OT-000003` |
| **Estación** | Cada etapa de producción. Son 7: Corte, Costura, Acabado, Control de calidad, Planchado, Empaque y Entrega |
| **Ficha viajera** | Hoja impresa que acompaña a las prendas por el taller, con el código de la orden. También da nombre al sistema visual |
| **Avance** | Marcar que una orden terminó su estación actual y pasa a la siguiente |
| **Evidencia** | Foto que se sube en una estación para mostrar el estado de la prenda |
| **Semáforo** | Estado de entrega de la orden: a tiempo, en riesgo o atrasada |
| **Link de seguimiento** | Enlace privado que el taller comparte con su cliente para que vea su orden |
| **Multi-tenant** | Un solo sistema que atiende a muchos talleres, con los datos de cada uno aislados |
| **SaaS** | *Software as a Service*: software que se usa por internet pagando una suscripción, sin instalar nada |
