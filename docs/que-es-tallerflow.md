# TallerFlow: resumen del proyecto

> **Cada orden visible, de Corte a Entrega.**

**Actualizado:** 21 de octubre de 2026, día 1 del Sprint 4.
**Responsable del documento:** José (líder técnico).
**Para quién:** el equipo, los docentes del curso y cualquier persona que necesite entender el proyecto en 10 minutos.

## 1. Resumen ejecutivo

TallerFlow es una plataforma web y móvil para **talleres de confección** que permite seguir cada pedido desde que entra al taller hasta que se entrega. Nace en Gamarra (Lima), donde los talleres coordinan su producción por WhatsApp, llamadas, cuadernos y Excel.

| Indicador | Estado |
|---|---|
| Producto | El MVP funciona: órdenes, 7 estaciones, fotos, escáner, equipo y link para el cliente |
| Producción | En línea en `https://tallerflowpe.com` desde el 20 de octubre de 2026 |
| Avance | 3 sprints cerrados de 4 planificados para el curso. El Sprint 4 va del 21 de octubre al 3 de noviembre |
| Meta del Sprint 4 | Que cualquier taller real cree su cuenta, verifique su correo y trabaje desde la web o desde Android |
| Mayor riesgo hoy | El registro abierto de talleres es el camino crítico: sin él, la landing no convierte y la demo no se puede hacer |
| Costo de operación | Entre S/ 85 y S/ 100 al mes para el piloto |

## 2. El problema y la propuesta de valor

En un taller típico, la información de cada pedido está repartida en mensajes, papeles y la memoria de las personas. Eso causa:

- **Falta de visibilidad:** el dueño no sabe qué pedido está atrasado ni en qué estación está, hasta que pregunta.
- **Retrasos descubiertos tarde:** el atraso se nota el día de la entrega, cuando ya no hay margen.
- **Llamadas constantes:** el cliente pregunta "¿cómo va mi pedido?" y alguien deja de producir para responder.
- **Doble registro y errores:** el cuaderno no coincide con lo que se dijo por WhatsApp.
- **Sin evidencia:** ante un reclamo, no hay cómo mostrar el estado de la prenda en cada etapa.

TallerFlow reemplaza esa información dispersa por **una sola fuente de verdad compartida**:

| Para quién | Qué le da TallerFlow |
|---|---|
| **Dueño o administrador** | Un panel que dice de un vistazo cuántas órdenes van atrasadas, cuáles salen hoy y cuánta carga tiene cada estación |
| **Operario** | Una pantalla simple en el celular para pasar la orden a la siguiente estación, subir una foto y escanear la ficha |
| **Cliente final** | Un link privado con la estación actual, la fecha prometida y las fotos del avance, sin instalar nada |

**Segmentos:**

| Perfil | Descripción | Rol en el negocio |
|---|---|---|
| Taller en crecimiento | De 10 a 30 operarios, varias órdenes en paralelo | **Cliente objetivo inicial.** Sufre el desorden a diario y está más dispuesto a pagar |
| Taller tradicional | De 5 a 15 operarios, con WhatsApp y cuaderno | Cliente pagador potencial; necesita algo simple, no un ERP |
| Marca que terceriza | Marca de moda, mayorista o retail | No paga al inicio, pero exige transparencia e impulsa la adopción |

## 3. Objetivos y criterios de éxito

**Objetivo del proyecto (curso, Ciclo IX):** dejar un SaaS en producción que talleres reales puedan usar solos, con datos aislados por taller, respaldos y monitoreo, listo para un piloto.

**Criterios de éxito al cierre del Sprint 4** (definición de terminado del plan):

- [ ] Un taller nuevo se registra en `/registro`, recibe el correo de verificación en Gmail y Outlook (no en spam) y entra a "Primeros pasos".
- [ ] Sin verificar el correo no se puede entrar, y dos talleres nuevos no ven datos uno del otro.
- [ ] La base rechaza dos estaciones en proceso. Un avance por error se deshace en un toque y una orden se puede cancelar.
- [ ] El dueño edita una orden y descarga sus órdenes y clientes en CSV.
- [ ] La consola de plataforma suspende un taller y eso saca a sus miembros de la web y de la app.
- [ ] Hay un respaldo diario fuera del VPS y una restauración probada, y el monitoreo avisa si el sitio se cae.
- [ ] La app Android firmada se instala desde la landing y funciona contra producción.
- [ ] El CI está en verde en `main`.

**Métricas del piloto** (propuestas, se miden desde el Sprint 5):

| Métrica | Qué responde |
|---|---|
| Talleres que completan el registro y crean su primera orden | ¿El producto se entiende sin ayuda? |
| Órdenes con avance registrado en cada estación | ¿El operario realmente lo usa? |
| Links de seguimiento abiertos por cliente | ¿Reemplaza las llamadas de "¿cómo va mi pedido?"? |
| Órdenes entregadas a tiempo, antes y después | ¿Reduce los retrasos? |
| Disponibilidad mensual del sitio | ¿Es confiable para operar todos los días? |

## 4. Alcance

**Dentro del proyecto (hasta el Sprint 4):**

- Gestión de órdenes por 7 estaciones, con fotos de evidencia, escáner y ficha imprimible.
- Clientes, equipo con roles e invitaciones por correo, y recuperación de contraseña.
- Link de seguimiento para el cliente final.
- Registro abierto de talleres con verificación de correo.
- App Android con el mismo código que la web.
- Consola de plataforma para suspender y eliminar talleres.
- Producción con dominio propio, respaldos, monitoreo y despliegue automatizado.

**Fuera del proyecto por ahora (Sprint 5 en adelante):**

| Tema | Por qué esperar |
|---|---|
| Planes y cobro | Primero hay que validar con el piloto cuánto pagarían los talleres. La suspensión desde la consola ya permite cortar por falta de pago |
| App iOS | Necesita una Mac y la cuenta de Apple Developer |
| Trabajar sin internet | Primero hay que medir cuánto falla la señal en los talleres |
| Notificaciones push | Necesita Firebase y decidir qué se avisa |
| Código QR impreso en la ficha | El escáner ya lee QR; falta generarlo en la ficha |
| Varios talleres por persona | Cambia el modelo de usuario y agrega un selector de taller |
| Acceso de soporte con permiso del dueño, página de estado pública, aprobación manual de talleres | Se priorizan según lo que pida el piloto |

## 5. Cronograma e hitos

| Etapa | Fechas | Meta | Estado |
|---|---|---|---|
| **Sprint 1: cimientos seguros** | 21 al 26 de setiembre | Base con aislamiento por taller, autenticación, roles, Docker y CI | ✅ Cerrado |
| **Sprint 2: operación del taller** | 27 de setiembre al 6 de octubre | Clientes, órdenes, 7 estaciones, fotos, escáner, equipo, recuperación de contraseña, link del cliente y sistema visual | ✅ Cerrado |
| **Sprint 3: salir a producción** | 17 al 20 de octubre (planificado del 19 al 30) | Dominio propio, correo real, marca, landing, app Android y sesión móvil | ✅ Cerrado con alcance parcial; lo pendiente pasó al Sprint 4 |
| **Sprint 4: talleres reales en producción** | 21 de octubre al 3 de noviembre | Registro abierto, órdenes sin miedo a equivocarse, consola, respaldos, monitoreo y APK firmado | 🟡 En curso (día 1) |
| **Sprint 5: piloto y cobro** | Por definir | Talleres piloto, métricas, planes y cobro | 📋 Backlog |

**Hitos ya logrados:**

- 26 de setiembre: primer recorrido completo de punta a punta (Sprint 1).
- 6 de octubre: operación completa de órdenes integrada (Sprint 2).
- 20 de octubre: **TallerFlow en producción** en `tallerflowpe.com`, con correo verificado (DKIM, SPF y DMARC) y la etiqueta `sprint-3` en `main`.
- 20 de octubre: respaldos cifrados en Cloudflare R2 y servidor corriendo `main`.

**Qué pasó del Sprint 3 al Sprint 4:** registro abierto y verificación de correo, deshacer avance, cancelar orden, una sola estación en proceso, consola de plataforma, respaldos y monitoreo instalados, y APK firmado. El Sprint 3 se cerró antes para poner producción en marcha. Lo que no alcanzó se reordenó en el Sprint 4 sin cambiar la meta.

## 6. Estado del Sprint 4

**Calendario:**

| Días | Fechas | José | Lucero | Michael | Stephano |
|---|---|---|---|---|---|
| 1–2 | 21 y 22 oct | Producción en `main`; endpoints de registro | Dominio real en los textos; entregabilidad | Subir su trabajo; una sola estación en proceso | CI verde; respaldos y restauración probada |
| 3–5 | 23 al 27 oct | `/registro`, verificación y antiabuso | Correo de verificación; editar orden (vista) | Deshacer avance; cancelar orden | Monitoreo; despliegue con un clic |
| 6–8 | 28 al 30 oct | Taller suspendido en el login; pendientes técnicos | Exportar datos | Editar orden (backend); consola | APK firmado; fotos en R2 |
| 9 | 2 nov | Fusionar todo en orden | Link de la app en la landing | E2E de la consola | Prueba en celulares reales |
| 10 | 3 nov | Verificación final y **demo** | Correcciones | Correcciones | Revisar cabeceras en producción |

**Avance por frente:**

| Frente | Responsable | Estado | Detalle |
|---|---|---|---|
| Producción en `main` | José | ✅ Hecho | El VPS corre `main`, con respaldo previo y verificación de salud |
| CI en `main` | José y Stephano | 🔴 Roto | El job `sprint-2` falla de forma intermitente. La corrección está en el PR `fix/s4-jose-ci-flyway-wait` |
| CI de Android | Stephano | ✅ Hecho | Runner fijado a Ubuntu 24.04 (#6) |
| Respaldos | Stephano | 🟡 En curso | Scripts integrados (#4) y credenciales de R2 en el servidor. Falta activar el respaldo diario y probar una restauración |
| Monitoreo | Stephano | 🟡 En curso | `/health/ready` responde a los monitores. Falta configurar UptimeRobot y la alerta de disco |
| Registro abierto y verificación | José | ⏳ Por iniciar | Contrato y migración `V11` listos desde el Sprint 3 |
| Dominio real en los textos | Lucero | 🟡 En curso | Rama `feat/s4-lucero-domain-texts` subida |
| Órdenes y consola | Michael | 🟡 En curso | Rama `feat/s4-luis-michael-clean` subida, con `V14` a `V16` |
| Marca "Etiqueta tejida" | José | 🟡 En revisión | PR `feat/s4-jose-rebrand`: logo, colores, landing, correos e íconos |
| Seguridad | José | 🟡 En curso | Revisión interna hecha el 20 de octubre; correcciones planificadas en este sprint |

**Orden de fusión acordado:** Stephano (respaldos) → Michael (órdenes) → José (registro) → Lucero (editar y exportar) → Michael (consola) → Stephano (fotos en R2) → integración final de José.

## 7. Equipo y responsabilidades

| Persona | Rol en el proyecto | Sprint 4 |
|---|---|---|
| **José** | Líder técnico: autenticación, registro, producción, integración y revisión de PR | Registro abierto, verificación de correo, antiabuso, taller suspendido en el login, despliegues e integración final |
| **Lucero** | Frontend, experiencia de usuario, correos y marca | Dominio real, entregabilidad del correo, editar orden (vista), exportar datos |
| **Michael** | Multi-tenancy, talleres y roles, órdenes y consola | Una sola estación en proceso, deshacer, cancelar, editar orden (backend), consola de plataforma |
| **Stephano** | Infraestructura, CI/CD, base de datos y app móvil | Respaldos, monitoreo, despliegue con un clic, APK firmado, fotos en R2 |

**Revisión cruzada de PR:** Michael revisa a José; José revisa a Lucero, Michael y Stephano.

**Dueños de archivos compartidos:** Stephano (Compose, CI y roles de la base), José (Caddy, túnel y `app.go`), Lucero (estilos). El detalle está en el plan del sprint.

## 8. Riesgos

| # | Riesgo | Prob. | Impacto | Mitigación | Dueño |
|---|---|---|---|---|---|
| R1 | El CI roto en `main` bloquea todas las fusiones del sprint | Alta | Alto | Fusionar primero `fix/s4-jose-ci-flyway-wait` y confirmar que el CI quede en verde | José |
| R2 | Choque de migraciones: la rama de Michael usa `V16`, reservada para las fotos en R2 de Stephano | Alta | Medio | Decidir antes de fusionar. Lo más simple: la de Michael queda en `V16` y la de R2 pasa a `V17` | José, Michael y Stephano |
| R3 | El registro abierto es el camino crítico de la meta, la landing y la demo | Media | Alto | Empezar el día 1 con el contrato congelado; `TF_SIGNUP_ENABLED` permite cerrarlo si algo falla | José |
| R4 | Conocimiento y accesos concentrados en una persona: producción, integración y la mayoría de las revisiones | Alta | Alto | Runbooks en `docs/runbooks/`, despliegue con un clic y una segunda persona con acceso de emergencia documentado | José |
| R5 | Respaldos sin una restauración probada | Media | Alto | Simulacro de restauración en este sprint, anotando cuánto tarda | Stephano |
| R6 | Abrir el registro antes de cerrar los hallazgos de la revisión de seguridad | Media | Alto | Los hallazgos de prioridad media se corrigen antes de publicar `/registro`. El reporte es privado y no se sube al repositorio | José |
| R7 | El correo de verificación cae en spam o no llega a Outlook | Media | Alto | Prueba en mail-tester (9/10 o más), Gmail y Outlook | Lucero |
| R8 | Lo que arrastra el Sprint 3 (órdenes y consola) no entra completo | Media | Medio | Plan B: primero una sola estación, deshacer y cancelar; la consola entra con solo suspender y reactivar | Michael |
| R9 | No hay datos de uso real para fijar el precio | Media | Alto | Piloto con métricas en el Sprint 5 antes de diseñar planes y cobro | Equipo |

## 9. Dependencias externas y costos

| Servicio | Para qué | Costo aproximado |
|---|---|---|
| VPS Contabo (Ubuntu 24.04) | Servidor de producción | Parte principal del presupuesto mensual |
| Cloudflare | Dominio `tallerflowpe.com`, DNS, túnel y almacenamiento R2 para respaldos | Dominio anual; R2 dentro del nivel gratuito en el piloto |
| Resend | Envío de correos (recuperación, invitaciones y verificación) | Nivel gratuito en el piloto |
| GitHub | Código, PR y CI | Gratuito |
| UptimeRobot | Monitoreo del sitio | Gratuito |

**Total del piloto:** entre S/ 85 y S/ 100 al mes.

## 10. Decisiones clave

| Fecha | Decisión | Por qué |
|---|---|---|
| 21 set | Go, Vue 3 y PostgreSQL, con el aislamiento entre talleres dentro de la base (Row-Level Security) | Aunque haya un error en el código, la base no entrega datos de otro taller |
| Sprint 2 | Diseño pensado primero para el celular | El operario trabaja de pie, con el celular y a veces con guantes |
| 17 oct | App Android con Capacitor, el mismo código que la web | Un solo equipo pequeño mantiene web y app a la vez |
| 20 oct | VPS detrás de Cloudflare Tunnel, con solo SSH expuesto | El servidor no publica puertos web a internet |
| 20 oct | Correo con Resend y dominio verificado | Los correos llegan a la bandeja de entrada, no a spam |
| 20 oct | Respaldos cifrados con `age` en Cloudflare R2 | Los respaldos viven fuera del servidor y nadie los lee sin la llave |
| 20 oct | Marca "Etiqueta tejida" | Logo y colores definitivos del equipo; reemplaza la identidad provisional del Sprint 2 |
| 21 oct | Planes y cobro pasan al Sprint 5 | Primero se valida el precio con el piloto |

## 11. Cómo trabajamos

- **Ramas:** cada entregable sale de una rama nueva desde `main` (`feat/s4-<persona>-<entregable>`). Nadie trabaja directo sobre `main`.
- **PR:** todo entra por PR con el CI en verde y una revisión. Si cambia la API, el contrato OpenAPI se actualiza antes que el código.
- **Migraciones:** números reservados por persona. `V12` queda vacía para siempre, y una migración ya fusionada nunca se edita.
- **Despliegue:** José despliega cada día en que entra algo a `main`. Nunca se despliega una migración sin un respaldo del mismo día.
- **Secretos:** solo en el servidor. Nunca en el repositorio ni en el chat del grupo.
- **Reunión diaria de 15 minutos:** qué terminé, qué hago hoy, qué me bloquea. Un bloqueo de más de medio día se avisa apenas aparece.

## 12. El producto en breve

**La vida de una orden:**

```text
 1. Recepción      El dueño registra la orden: cliente, prenda, cantidad, fecha prometida.
        │          TallerFlow le asigna un código (OT-000003).
        ▼
 2. Ficha viajera  Se imprime la ficha con el código y viaja con las prendas por el taller.
        ▼
 3. Producción     Corte → Costura → Acabado → Control de calidad → Planchado → Empaque → Entrega
        │          En cada estación el operario escanea la ficha, toca "Pasar a…" y sube una foto.
        ▼
 4. Seguimiento    El cliente abre su link: "Tu pedido está en Costura", 3 de 7, entrega el viernes.
        ▼
 5. Entrega        Al completar Entrega, la orden pasa a Completadas.
```

**Semáforo de entrega:** **a tiempo** (verde, ●), **en riesgo** (ámbar, ▲) y **atrasada** (rojo, ■!, el único rojo sólido de la app). El estado siempre lleva una palabra y una forma, no solo un color.

**Quién puede hacer qué:**

| Nivel | Quién | Qué ve y hace |
|---|---|---|
| Propietario (OWNER) | El dueño del taller | Todo: panel, órdenes, clientes, equipo y corrección de avances |
| Administrador (ADMIN) | Quien el dueño designe | Lo mismo que el propietario |
| Operario (OPERATOR) | Personal de producción | Solo órdenes: avanzar estaciones, subir fotos y escanear |
| Cliente final | El cliente del taller, sin cuenta | Una sola orden, con su link privado |
| Plataforma *(Sprint 4)* | Los dueños de TallerFlow | Lista de talleres; suspender, reactivar y eliminar. **No** ve órdenes ni clientes |

**Seguridad:**

- Cada taller ve solo sus datos: el aislamiento lo aplica PostgreSQL.
- Contraseñas con Argon2id.
- Sesiones revocables con protección CSRF y límite de intentos.
- Links de seguimiento con tokens que solo se guardan cifrados.
- Auditoría de las acciones sensibles.

**Diseño, "Etiqueta tejida"** (definido en `DESIGN.md`):

- **Logo:** un rombo tejido de 4×4.
- **Colores:** marino `#0A346F` como color de marca y turquesa `#00BFB2` solo como hilo de firma, nunca para indicar un estado.
- **Fichas:** las órdenes se ven como fichas con una costura de 7 puntadas que se cose a medida que la orden avanza.
- **Uso en el taller:** botones de al menos 44 px, todo usable con teclado y contraste suficiente para leer bajo la luz del taller.

**Cómo está construido:**

```text
  Navegador / App Android ──▶ Cloudflare ──▶ Túnel ──▶ VPS: Caddy ──▶ API Go ──▶ PostgreSQL
                                                              └──▶ nginx (web Vue)
```

| Capa | Tecnología |
|---|---|
| Web y app | Vue 3 + TypeScript; Android con Capacitor |
| API | Go (Gin y GORM) |
| Base de datos | PostgreSQL 18 con Row-Level Security y Flyway |
| Infraestructura | Docker Compose en un VPS, detrás de Cloudflare Tunnel |
| Calidad | Pruebas de Go, Vitest, Playwright a 360 px, pruebas SQL de seguridad y GitHub Actions en cada PR |

## 13. Modelo de negocio (propuesto)

- **SaaS con suscripción mensual:** el taller no instala nada y paga por visibilidad, menos llamadas y clientes más satisfechos.
- **Un solo sistema para todos los talleres:** mantiene bajo el costo por taller.
- **Crecimiento:** cada link de seguimiento que el taller comparte muestra la marca TallerFlow a clientes que también trabajan con otros talleres.
- **Precio:** se define con lo que se aprenda en el piloto del Sprint 5.

## 14. Dónde está el detalle

| Documento | Contenido |
|---|---|
| `docs/sprints/sprint-04-team-plan.md` | Plan del sprint en curso: entregables, contratos, calendario y definición de terminado |
| `docs/sprints/sprint-03-team-plan.md` | Diseño detallado de registro, deshacer, cancelar y consola |
| `docs/contexto-proyecto.md` | Idea original, perfiles de usuario y visión |
| `DESIGN.md` | Sistema visual |
| `docs/contracts/openapi.yaml` | Contratos de la API |
| `docs/runbooks/` | Servidor, despliegue, respaldos, monitoreo y Android |
| `docs/reviews/` | Revisiones de UX y de marca |

## 15. Glosario

| Término | Significado |
|---|---|
| **Orden (OT)** | Un pedido de confección: cliente, prenda, cantidad y fecha prometida, con un código como `OT-000003` |
| **Estación** | Cada etapa de producción. Son 7: Corte, Costura, Acabado, Control de calidad, Planchado, Empaque y Entrega |
| **Ficha viajera** | Hoja impresa que acompaña a las prendas por el taller, con el código de la orden |
| **Avance** | Marcar que una orden terminó su estación actual y pasa a la siguiente |
| **Semáforo** | Estado de entrega de la orden: a tiempo, en riesgo o atrasada |
| **Link de seguimiento** | Enlace privado que el taller comparte con su cliente para que vea su orden |
| **Multi-tenant** | Un solo sistema que atiende a muchos talleres, con los datos de cada uno aislados |
| **SaaS** | Software que se usa por internet con una suscripción, sin instalar nada |
| **Sprint** | Bloque de trabajo de unos 10 días hábiles con una meta y una demo al final |
| **PR** | *Pull request*: propuesta de cambio que se revisa y prueba antes de entrar a `main` |
| **CI** | Pruebas automáticas que corren en GitHub en cada PR |
