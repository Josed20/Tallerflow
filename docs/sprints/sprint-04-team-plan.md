# TallerFlow — Sprint 4: talleres reales en producción

**Duración:** 10 días hábiles, del miércoles 21 de octubre al martes 3 de noviembre de 2026.

**Punto de partida:** la etiqueta `sprint-3` en `main` (commit `af3ba23`). Todo lo que el equipo entregó en el Sprint 3 ya está ahí, probado de punta a punta.

**Meta:**

- Cualquier taller crea su cuenta en `https://tallerflowpe.com`, verifica su correo y empieza a trabajar desde la web o desde la app Android.
- Un avance por error se deshace en un toque, una orden se puede cancelar y la base nunca acepta dos estaciones en proceso.
- Los dueños de TallerFlow tienen una consola para ver, suspender y eliminar talleres.
- Producción tiene respaldos diarios probados, monitoreo y despliegue con un clic.

**Equipo:**

| Persona | Responsabilidad |
|---|---|
| José | Registro abierto y verificación de correo, taller suspendido en el login, producción en `main`, integración final |
| Lucero | Correo en producción, textos con el dominio real, editar órdenes (vista), exportar datos del taller |
| Michael | Órdenes sin miedo a equivocarse (una sola estación en proceso, deshacer, cancelar) y consola de plataforma |
| Stephano | Respaldos y monitoreo en producción, despliegue automático, APK firmado y fotos en Cloudflare R2 |

**Fuentes de verdad:**

- `CLAUDE.md`: comandos y arquitectura.
- `DESIGN.md`: sistema visual "Ficha viajera".
- `docs/contracts/openapi.yaml`: contratos de la API.
- `docs/runbooks/`: servidor, despliegue, respaldos, monitoreo y Android.
- `docs/sprints/sprint-03-team-plan.md`: el detalle de las piezas que pasan a este sprint.

## 1. Dónde estamos hoy

| Área | Estado | Qué falta |
|---|---|---|
| Producción | `https://tallerflowpe.com` corre en el VPS (Contabo, Ubuntu 24.04) detrás de Cloudflare Tunnel. Solo SSH abierto | Pasar el servidor a `main` (hoy corre la rama de despliegue, base en V10) |
| Correo | Resend con el dominio verificado (DKIM, SPF y DMARC). La recuperación de contraseña llega a la bandeja de Gmail | Prueba de entregabilidad y Outlook |
| Marca y páginas públicas | Logo, íconos, vista previa para WhatsApp, landing, términos, privacidad y "Primeros pasos" | Link real de descarga de la app |
| Plantillas de correo | Las cinco del contrato 5.1, con marca | Que José y Michael las usen |
| App Android | Capacitor con cámara, escáner, token en el Keystore y cliente HTTP con Bearer | APK firmado contra producción |
| Sesión móvil | Bearer, CORS móvil y `V13` | Nada |
| Registro abierto | Solo la base: `V11` y el contrato OpenAPI | Los endpoints, `/registro`, el antiabuso y `EMAIL_NOT_VERIFIED` |
| Órdenes | Igual que en el Sprint 2 | Una sola estación en proceso, deshacer, cancelar y editar |
| Consola de plataforma | No existe | Todo (sección 7.4 del plan del Sprint 3) |
| Respaldos y monitoreo | Scripts y runbooks en el repo | Instalarlos en el VPS y probar una restauración |
| CI | La suite completa pasa (incluido `-race`) | El job `android-debug` falla al preparar el SDK de Android |

## 2. Cómo trabajamos con las ramas

`main` es la única base. Cada entregable sale de una rama nueva creada desde `main` actualizado:

```bash
git fetch --all --prune
git switch main
git pull --ff-only
git switch -c feat/s4-<persona>-<entregable>
```

Por ejemplo: `feat/s4-michael-single-stage`, `feat/s4-jose-signup`, `feat/s4-stephano-backups`.

- **Un PR por entregable.** Cuando el PR se fusiona, esa rama se deja de usar y el siguiente entregable sale de otra rama nueva desde `main`. Seguir trabajando en una rama ya fusionada repite commits en el siguiente PR.
- **Todos los días:** `git pull --rebase origin main` en la rama en la que estés.
- **Nadie trabaja ni hace push directo a `main`.** Todo entra por PR con el CI verde y una revisión.
- **Ramas del Sprint 3:** no se vuelven a usar. Cada uno borra las suyas cuando confirme que su trabajo está en `main` (`git push origin --delete <rama>`). `feat/s3-stephano-brand-identity` quedó fuera a propósito: está obsoleta y su logo lo reemplazó el paquete de marca.
- Nunca subir contraseñas, llaves de API, `.env`, keystores de Android ni credenciales al repo, ni pegarlos en el chat del grupo.

### Archivos compartidos

Siguen las reglas del Sprint 3:

| Archivo | Regla |
|---|---|
| `backend/cmd/api/app.go` | Cada uno agrega solo las líneas de su módulo; José resuelve conflictos al fusionar |
| `backend/platform/config/config.go` | Cada uno agrega sus variables `TF_*`, siempre con valor por defecto para desarrollo |
| `frontend/src/app/router.ts` | Cada uno agrega sus rutas en un bloque `// Sprint 4 - <persona>` |
| `docs/contracts/openapi.yaml` | El contrato se actualiza **antes** que el código |
| `compose*.yaml`, `.env.example`, `infra/postgres/init-roles.sh`, `.github/workflows/` | Stephano |
| `infra/caddy/Caddyfile`, `infra/cloudflared/config.yml` | José |
| `frontend/src/styles/` | Solo Lucero |

Una variable nueva que sea obligatoria en producción también va en `compose.production.yaml`. Además hay que avisarle a José para que la agregue al `.env` del VPS **antes** de desplegar.

### Migraciones Flyway: números reservados

`main` ya tiene `V11` y `V13`. **`V12` queda vacío para siempre:** Flyway rechaza una migración con un número menor que una ya aplicada, así que nadie la usa ni la rellena.

| Número | Dueño | Contenido |
|---|---|---|
| `V14` | Michael | Una sola estación en proceso, `completed_by` en las etapas y campos de cancelación (lo que en el Sprint 3 era `V12`) |
| `V15` | Michael | Estado del taller, tablas de la consola y sus funciones SQL (lo que en el Sprint 3 era `V14`) |
| `V16` | Stephano | Fotos en R2 (clave del objeto en `stage_photos`) |
| `V17+` | Pedir a José | Cualquier otra necesidad |

**Michael:** si ya escribiste tus migraciones como `V12` y `V14`, renómbralas a `V14` y `V15` antes de subirlas. Nunca editar una migración ya fusionada. Si tu base local se desordena, recréala con `docker compose ... down --volumes`.

## 3. Producción

- **Servidor:** `ssh -i ~/.ssh/<tu-llave> deploy@<IP>`. José da acceso agregando tu llave **pública**; nunca se comparte la privada.
- **Carpeta:** `/opt/tallerflow`. Los secretos viven solo en `/opt/tallerflow/.env` y en `/etc/tallerflow/`.
- **Comando de Compose:** siempre los tres archivos:

  ```bash
  docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml --profile app ...
  ```

  Sin `compose.tunnel.yaml`, Caddy vuelve a publicar el puerto 8080 en HTTP, y Docker se salta el firewall para los puertos que publica.
- **Datos reales:** producción ya tiene talleres. Nadie prueba ahí: para eso está el entorno local con Mailpit.
- **No correr el túnel `tallerflow` en una PC:** Cloudflare repartiría las visitas entre esa PC y el servidor.

## 4. Contratos congelados el día 1

### 4.1 Registro y verificación (José implementa; Lucero y Michael lo usan)

Ya están en `docs/contracts/openapi.yaml`:

- `POST /api/v1/signup` responde siempre `202`, exista o no el correo.
- `POST /api/v1/auth/email-verifications/consume` verifica e inicia sesión web.
- `POST /api/v1/auth/email-verifications/resend` responde siempre `202`.
- Login con correo sin verificar: `403 EMAIL_NOT_VERIFIED`.
- El link del correo es `https://tallerflowpe.com/verificar-correo?token=...` y usa la plantilla `verify-email`.

### 4.2 Estado del taller (Michael implementa; José lo muestra en el login)

Es el mismo contrato 5.7 del Sprint 3, ahora en `V15`:

- `workshops.status`: `ACTIVE`, `SUSPENDED` o `PENDING_DELETION`.
- `workshops.Service.Resolve` devuelve `ErrWorkshopSuspended` si el taller no está `ACTIVE`. Eso corta la sesión web y la móvil.
- José lo mapea a `403 WORKSHOP_SUSPENDED`. La app Android ya muestra ese mensaje.

### 4.3 Editar orden (Michael el backend; Lucero la vista)

- `PATCH /api/v1/orders/:id`: prenda, cantidad, cliente y fecha prometida. Solo OWNER y ADMIN.
- Usa `orders.version` para que dos ediciones a la vez no se pisen: si no coincide, responde `409 ORDER_VERSION_CONFLICT`.
- No permite editar órdenes `COMPLETED` ni `CANCELLED`.
- Registra `ORDER_UPDATED` en la auditoría, con los campos que cambiaron.

### 4.4 Exportar datos (Lucero)

- `GET /api/v1/exports/orders.csv` y `GET /api/v1/exports/clients.csv`. Solo OWNER y ADMIN, siempre dentro de `WithinTenant`.
- CSV en UTF-8 con BOM, para que Excel muestre bien las tildes, y separado por comas.

### 4.5 Fotos en R2 (Stephano)

- Las fotos nuevas se suben a un bucket privado de Cloudflare R2. La base guarda solo la clave del objeto.
- Se muestran con URLs firmadas que vencen en minutos. El link del cliente usa el mismo mecanismo.
- Las fotos que ya están en la base se siguen mostrando. No hace falta migrarlas en este sprint.

## 5. José — Registro abierto, producción e integración

**Objetivo:** que cualquier taller se registre solo, y que producción corra siempre lo que está en `main`.

**Entregables:**

1. **Producción en `main` (días 1 y 2).** Pasar el VPS de la rama de despliegue a `main`. Con eso se aplican `V11` y `V13`, y se publican la landing y la sesión móvil. Antes, sacar un snapshot del VPS y un `pg_dump`. Después, comprobar el login web, la recuperación de contraseña y `/health/ready`.
2. **Registro abierto (`/registro`).** Crea el usuario, el taller `ACTIVE` y la membresía OWNER en una sola transacción, con aceptación de términos. Cuando el registro abierto esté en producción, `/onboarding` deja de ser público. El detalle está en la sección 6 del plan del Sprint 3.
3. **Verificación de correo.** Los endpoints `consume` y `resend`, la vista `/verificar-correo`, `403 EMAIL_NOT_VERIFIED` y el botón para reenviar en `LoginView`. El CLI de bootstrap y la aceptación de invitaciones marcan `email_verified_at`, porque esos correos ya se comprobaron.
4. **Antiabuso.** Rate limit por IP en el registro y el reenvío, el mismo mensaje exista o no el correo, y `TF_SIGNUP_ENABLED=false` para cerrar el registro.
5. **Taller suspendido en el login** (contrato 4.2).
6. **Pendientes técnicos:**
   - Gin en modo `release` en producción (hoy imprime `[GIN-debug]`).
   - Registrar en la auditoría los inicios de sesión `MOBILE`.
7. **`scripts/verify-sprint4.*`**, partiendo del de Sprint 2: suma los tests SQL nuevos (`email_verification_contract.sql` y los de Michael) y los E2E nuevos.
8. **Integración (días 9 y 10)** y la demo.

**Pruebas mínimas:**

- Dos talleres registrados no ven datos uno del otro.
- Un token de verificación vencido, usado o adulterado responde error.
- E2E `signup-flow.spec.ts`: registro → correo en Mailpit → verificar → panel con "Primeros pasos".

## 6. Michael — Órdenes y consola de plataforma

**Objetivo:** el mismo del Sprint 3, que pasa completo a este sprint. El detalle de cada pieza está en la sección 7 del plan del Sprint 3. Lo único que cambia son los números de migración.

**Entregables, en este orden:**

1. **Día 1: subir a GitHub lo que tengas, aunque esté a medias, como PR en borrador.** Así el equipo ve el avance y José puede integrarlo.
2. **Una sola estación en proceso (`V14`).** Reproducir con una prueba el caso de OT-000001, arreglarlo, reparar los datos y agregar el índice `order_stages_one_in_progress_idx`.
3. **Deshacer un avance por error** (sección 7.2 del Sprint 3). Ahora también desde la app: el endpoint acepta Bearer, igual que el resto de la API.
4. **Cancelar orden** (sección 7.3 del Sprint 3).
5. **Editar orden: backend** (contrato 4.3).
6. **Consola de plataforma (`V15`)** (sección 7.4 del Sprint 3). Ya están listos:
   - el host `admin.tallerflowpe.com`;
   - `TF_CONSOLE_ORIGIN` y `TF_CONSOLE_DATABASE_URL` en el servidor;
   - el rol `tallerflow_console`;
   - las plantillas `workshop-suspended` y `workshop-deletion-scheduled`.

   Le pasas a José el bloque de Caddy y la regla del túnel para `admin.`.

**Plan B por si se atrasa:** si el día 6 la consola no está empezada, se entregan primero los puntos 2 a 4 y la consola pasa a la semana siguiente con solo suspender y reactivar. Eliminar definitivamente puede esperar; suspender no.

## 7. Lucero — Correo en producción, editar y exportar

**Objetivo:** que todo lo que llega al correo del taller se vea bien y llegue a la bandeja de entrada, y que el dueño pueda corregir y llevarse sus datos.

**Entregables:**

1. **Dominio real en todos los textos.** Reemplazar `tallerflow.pe` por `tallerflowpe.com`:
   - en las plantillas;
   - en la landing y los legales;
   - en el valor por defecto de `TF_SMTP_FROM` (`config.go` y `.env.example`);
   - en `soporte@...`.
2. **Entregabilidad.**
   - Prueba en mail-tester.com con 9/10 o más.
   - Revisar en Gmail (web y celular) y en Outlook.
   - Revisar que los links vayan directo a `tallerflowpe.com`, sin redirecciones de tracking.
   - Activar Cloudflare Email Routing para que `soporte@tallerflowpe.com` reenvíe a un buzón real.
3. **Correo de verificación con marca.** Coordinar con José que el registro use `verify-email` y probarlo de punta a punta en Mailpit.
4. **Editar orden: la vista** (contrato 4.3).
   - Formulario en el detalle de la orden, con los botones unificados.
   - Mensaje claro cuando otra persona la editó a la vez (`409`).
5. **Exportar datos del taller** (contrato 4.4). El endpoint y un botón "Descargar mis datos" en el panel o en el equipo, para OWNER y ADMIN.
6. **Landing:** el botón "Descargar app Android" apunta al link que pase Stephano.
7. **Si sobra tiempo:** modo oscuro con los tokens de `DESIGN.md`.

**Pruebas mínimas:**

- El CSV abre bien en Excel, con tildes y eñes.
- Un OPERATOR recibe `403` al exportar o editar.
- E2E: editar una orden y ver el cambio en el detalle y en el link del cliente.

## 8. Stephano — Producción confiable y app publicada

**Objetivo:** que no perdamos datos, que sepamos de una caída antes que los talleres y que la app se instale fácil.

**Entregables:**

0. **Día 1: CI verde en `main`.** El job `android-debug` falló en su primera ejecución, en el paso `android-actions/setup-android@v3`, antes de compilar. El job `sprint-2`, con `-race`, pasó. El log de GitHub (con la sesión iniciada) dice la causa exacta. Lo más probable es la migración de `ubuntu-latest` a Ubuntu 26, que empezó el 19 de octubre, junto con acciones ya obsoletas (`setup-android@v3`, `setup-java@v4`). Hay que actualizarlas o fijar el runner a `ubuntu-24.04`.
1. **Respaldos en el VPS (días 1 y 2: es lo más urgente del sprint).**
   - Instalar `age` y la CLI de AWS en el servidor.
   - Crear el bucket privado en Cloudflare R2. Recordar que `aws s3 cp` necesita `AWS_ENDPOINT_URL` de R2.
   - Programar `scripts/backup-vps.sh` a diario con un timer de systemd o cron, guardando 7 diarios y 4 semanales.
   - **Probar una restauración completa** en una base vacía con `scripts/restore-drill-local.sh` y anotar cuánto tardó.
2. **Monitoreo activo.** UptimeRobot sobre `https://tallerflowpe.com/health/ready` y la página de inicio, con aviso al grupo. Alerta si el disco del VPS pasa del 80%.
3. **Despliegue con un clic.** Configurar los secretos de GitHub (`DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_PATH`, `DEPLOY_KNOWN_HOSTS`, `DEPLOY_SSH_KEY`, `TF_HEALTH_URL`) con una llave **propia del workflow**, no la de una persona. Probarlo con un cambio pequeño. El script ya usa los tres archivos de Compose.
4. **APK firmado contra producción.**
   - `VITE_API_BASE_URL=https://tallerflowpe.com`.
   - Keystore guardado fuera del repo, en los secretos de GitHub y en una copia segura de José.
   - Distribución: APK directo o prueba interna de Play Console. Pasar el link a Lucero para la landing.
   - Probar en al menos dos celulares reales.
5. **Fotos en Cloudflare R2 (`V16`)** (contrato 4.5).
6. **Cabeceras y CSP en producción.** Revisar en el navegador que no haya bloqueos en el panel, las órdenes, el escáner, el link de seguimiento y la consola.

**Pruebas mínimas:**

- La restauración funciona y queda anotado el tiempo.
- Desde una red externa, solo el puerto 22 del VPS responde.
- La app inicia sesión contra producción, sobrevive a cerrarla y abrirla, toma una foto y escanea una ficha.

## 9. Calendario

| Días | José | Lucero | Michael | Stephano |
|---|---|---|---|---|
| 1–2 | Producción en `main`; endpoints de registro | Dominio real en los textos; entregabilidad | Subir su trabajo; `V14` y el arreglo de una sola estación | CI verde; respaldos en el VPS y restauración probada |
| 3–5 | `/registro`, verificación y antiabuso | Correo de verificación con marca; editar orden (vista) | Deshacer avance; cancelar orden | Monitoreo; despliegue con un clic |
| 6–8 | Taller suspendido en el login; pendientes técnicos | Exportar datos | Editar orden (backend); consola (`V15`) | APK firmado; fotos en R2 |
| 9 | Fusionar todo en orden | Link de la app en la landing | E2E de la consola | Prueba en celulares reales |
| 10 | Verificación final y demo | Corregir lo que salga | Corregir lo que salga | Revisar cabeceras y CSP en producción |

**Orden de fusión:** Stephano (respaldos) → Michael (`V14` y órdenes) → José (registro y verificación) → Lucero (editar y exportar) → Michael (consola, `V15`) → Stephano (R2, `V16`) → integración final de José.

**Despliegue:** José despliega a producción al final de cada día en que entre algo a `main`, después de que el CI esté verde. Nunca se despliega una migración sin un respaldo del mismo día.

**Reunión diaria de 15 minutos:** qué terminé, qué hago hoy, qué me bloquea. Un bloqueo de más de medio día se avisa apenas aparece.

## 10. Revisión de PR

| Autor | Revisa | Foco |
|---|---|---|
| José | Michael | Tokens, RLS al crear talleres, antiabuso |
| Lucero | José | Correos, CSV dentro de `WithinTenant`, accesibilidad, `DESIGN.md` |
| Michael | José | Reglas de deshacer, funciones `SECURITY DEFINER`, permisos del rol de consola, 2FA |
| Stephano | José | Secretos, keystore, URLs firmadas de R2, timers del VPS |

Checklist del autor:

```markdown
- [ ] La rama salió de main actualizado y es nueva para este entregable.
- [ ] Solo toqué archivos de mi sección (o lo acordé en el grupo).
- [ ] Actualicé OpenAPI antes del código si cambió la API.
- [ ] Escribí la prueba que falla antes del arreglo.
- [ ] npm --prefix frontend test, npm --prefix frontend run build y go -C backend test ./... pasan.
- [ ] Si agregué una migración, usé mi número reservado.
- [ ] Si agregué una variable obligatoria en producción, está en compose.production.yaml y se la pasé a José.
- [ ] No hay secretos, .env, keystores ni credenciales en el diff.
```

## 11. Definición de terminado

**Talleres:**

- [ ] Un taller nuevo se registra en `https://tallerflowpe.com/registro`, recibe el correo en Gmail y Outlook (no en spam), lo verifica y ve "Primeros pasos".
- [ ] Sin verificar no se puede entrar.
- [ ] Dos talleres nuevos no ven datos uno del otro.
- [ ] El dueño edita una orden y descarga sus órdenes y clientes en CSV.

**Órdenes:**

- [ ] La base rechaza dos estaciones en proceso.
- [ ] Un avance por error se deshace en un toque dentro de 5 minutos, desde la web y desde la app.
- [ ] Una orden se cancela; completar Entrega por error también se deshace.

**Consola:**

- [ ] Solo se entra pasando Cloudflare Access, contraseña y código 2FA.
- [ ] Suspender un taller saca a sus miembros en la web y en la app; reactivarlo les devuelve el acceso.

**Producción:**

- [ ] El VPS corre `main` y se despliega con el workflow, sin entrar por SSH.
- [ ] Hay respaldo diario fuera del VPS y una restauración probada.
- [ ] El monitoreo avisa al grupo si el sitio se cae.
- [ ] La app Android firmada se instala desde el link de la landing y funciona contra producción.
- [ ] `scripts/verify-sprint4` pasa desde volúmenes vacíos y el CI está verde en `main`.

## 12. Demo de cierre

1. Stephano muestra el monitoreo en verde y el respaldo de anoche en R2.
2. José registra un taller nuevo con un correo real, lo verifica y muestra "Primeros pasos".
3. Un operario instala la app desde la landing, avanza una orden **por error** y la deshace desde el aviso.
4. Lucero edita esa orden desde la web y descarga el CSV del taller.
5. Michael entra a `admin.tallerflowpe.com` con 2FA, suspende el taller y la app del operario lo saca con el mensaje de suspensión. Luego lo reactiva.
6. Stephano restaura el respaldo en una base vacía.

## 13. Fuera de este sprint (Sprint 5)

| Tema | Por qué esperar |
|---|---|
| Planes y cobro | Primero hay que validar con los talleres del piloto cuánto pagarían. La suspensión de la consola ya sirve para cortar por falta de pago |
| Límites por plan | Depende de lo anterior |
| Acceso de soporte con permiso del dueño | El dueño autoriza por un tiempo limitado que soporte vea su taller, con auditoría |
| Aprobación manual de talleres nuevos | Solo si aparece spam; mientras tanto, `TF_SIGNUP_ENABLED=false` y la suspensión cubren el caso |
| Código QR impreso en la ficha viajera | El escáner ya lee QR; falta generarlo en la ficha |
| App iOS | Necesita una Mac y la cuenta de Apple Developer |
| Trabajar sin internet | Primero hay que medir cuánto falla la señal en los talleres |
| Notificaciones push | Necesita Firebase y una decisión sobre qué avisar |
| Varios talleres por persona | Cambia el índice de un taller por usuario y agrega un selector de taller |
| Página de estado pública | Para avisar a los talleres cuando hay una caída o un mantenimiento |
