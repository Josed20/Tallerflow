# TallerFlow — Sprint 3: Listo para operar como SaaS, en web y celular

**Duración:** 10 días hábiles, del lunes 19 al viernes 30 de octubre de 2026.

**Meta:**

- Cualquier taller crea su cuenta con su correo, la verifica, invita a su equipo por correo y trabaja sus órdenes desde la web o desde la app Android.
- Si alguien avanza una orden por error, puede deshacerlo.
- Como dueños de TallerFlow, tenemos una consola para ver todos los talleres, suspenderlos y eliminarlos.
- Todo funciona en un dominio propio con HTTPS.

**Equipo:**

| Persona | Responsabilidad |
|---|---|
| José | Registro de talleres con verificación de correo, VPS y dominio, sesión para la app móvil, integración final |
| Lucero | Correos con marca y correo real, recuperación, invitaciones por correo, logo e íconos, landing, legales y primeros pasos |
| Michael | Consola de plataforma y ciclo de vida de los talleres, deshacer avances, error de dos estaciones en proceso, cancelar órdenes |
| Stephano | App Android con Capacitor (incluye cámara y escáner nativos), cabeceras de seguridad, CI/CD, respaldos y monitoreo |

**Fuentes de verdad:**

- `CLAUDE.md`: comandos y arquitectura.
- `DESIGN.md`: sistema visual "Ficha viajera".
- `docs/contracts/openapi.yaml`: contratos de la API.
- `docs/sprints/sprint-03-operations-ux.md`: operación de órdenes ya entregada.

## 1. Dónde estamos hoy

| Área | Estado | Qué falta |
|---|---|---|
| Multi-tenant (RLS por taller) | Hecho y probado | Nada en la base; el aislamiento ya funciona |
| Alta de talleres | Solo el **primer** taller puede registrarse en `/onboarding`; `bootstrap_state` bloquea los siguientes | Registro abierto para muchos talleres, con verificación por correo |
| Administración de la plataforma | No existe. Para ver, suspender o borrar un taller hay que entrar a la base a mano | Consola de plataforma para el dueño de TallerFlow |
| Ciclo de vida del taller | Un taller solo puede existir | Activo, suspendido, eliminación programada y eliminado |
| Recuperar contraseña | Hecho (backend, vistas y Mailpit) | Probarla con correo real y con la plantilla de la marca |
| Invitar al equipo | Funciona, pero el link se copia a mano | Enviar la invitación por correo |
| Avance por error | Solo OWNER/ADMIN pueden "Corregir avance", con motivo; el operario no puede hacer nada | Deshacer inmediato para quien se equivocó (ver 7.2) |
| Órdenes | Crear, avanzar, fotos, link de seguimiento del cliente | Cancelar y arreglar las dos estaciones "en proceso" a la vez de OT-000001 |
| Celular | La web funciona a 360 px | App Android instalable con cámara y escáner nativos |
| Publicación | Túnel temporal de Cloudflare desde una PC | Toda la app en un VPS con dominio propio y HTTPS |

## 2. Cómo queda la arquitectura

```text
  Navegador (web)            App Android (Capacitor)        Consola de plataforma
  https://tallerflow.pe      mismo código Vue en la APK     https://admin.tallerflow.pe
     │ cookie + CSRF            │ Authorization: Bearer        │ Cloudflare Access + login con 2FA
     └──────────────┬───────────┴──────────────┬───────────────┘
                    ▼                          ▼
              Cloudflare (DNS, HTTPS, protección, Access para admin.)
                    ▼
  ┌───────────────────────── VPS ────────────────────────────────────┐
  │  Caddy ── /api/* y /health/* ──▶ backend Go (Gin)                 │
  │    └──── todo lo demás ────────▶ nginx (build de Vue)             │
  │                                       │                           │
  │      PostgreSQL 18 con RLS ◀──────────┘                           │
  │      rol tallerflow_app:     datos de UN taller (RLS)             │
  │      rol tallerflow_console: solo funciones de la consola,        │
  │                              sin leer órdenes ni clientes         │
  └───────────────────────────────────────────────────────────────────┘
```

`tallerflow.pe` es un ejemplo; se usa el dominio que compre José.

### Cuatro niveles de acceso

| Nivel | Quién | Dónde entra | Qué puede hacer |
|---|---|---|---|
| **Plataforma** | Los dueños de TallerFlow (José y quien él autorice) | `admin.tallerflow.pe` | Ver la lista de talleres y sus métricas, suspender, reactivar y eliminar. **No** ve órdenes, clientes ni fotos |
| **Taller: OWNER / ADMIN** | El dueño del taller y sus administradores | `tallerflow.pe/app` o la app | Todo su taller: panel, órdenes, clientes, equipo |
| **Taller: OPERATOR** | El operario | `tallerflow.pe/app` o la app | Solo Órdenes: avanzar, deshacer su avance, subir fotos y escanear |
| **Cliente final** | El cliente del taller, sin cuenta | Link `/seguimiento/:token` | Ver una sola orden |

### Por qué la consola no ve las órdenes de los talleres

Así funcionan los SaaS serios, como Shopify, Slack o Notion:

- **Confianza:** el taller guarda los datos de **sus** clientes y sus precios. Si sabe que nosotros no los vemos, es más fácil que nos los confíe.
- **Ley 29733:** mientras menos datos personales manejemos, menos riesgo y menos obligaciones.
- **Si roban la cuenta de la consola**, el atacante puede suspender talleres, pero no llevarse los datos de los clientes.

La consola ve **metadatos**: nombre del taller, correo del dueño, fechas, cantidad de miembros, órdenes y fotos, y última actividad. Si un taller pide soporte, comparte pantalla. El "acceso de soporte con permiso del dueño" queda para el Sprint 4.

Esto se impone en la base de datos, no solo en el código. El rol `tallerflow_console` no tiene permisos sobre las tablas de los talleres: solo puede ejecutar unas pocas funciones SQL que devuelven conteos y metadatos (sección 7.4).

### Ciclo de vida de un taller

```text
  /registro ──▶ ACTIVE ◀──── Reactivar ────┐
                  │                         │
                  ├── Suspender (motivo) ──▶ SUSPENDED ── Programar eliminación ──┐
                  │                                                              ▼
                  └──────── Programar eliminación (motivo) ──────────────▶ PENDING_DELETION
                                                                                 │ 30 días
                                                                                 ▼
                                                     Eliminar definitivamente (escribir el
                                                     nombre del taller + código 2FA)
```

| Estado | Qué pasa |
|---|---|
| `ACTIVE` | Funciona normal |
| `SUSPENDED` | Nadie del taller puede entrar; se cierran todas sus sesiones (web y app). Los links de seguimiento de sus clientes dejan de funcionar. **No se borra nada.** Se usa por falta de pago, abuso o a pedido del dueño |
| `PENDING_DELETION` | Igual que suspendido, y además el dueño recibe un correo: "Tu taller se eliminará el (fecha)". Durante 30 días se puede cancelar |
| Eliminado | Se borran el taller, sus órdenes, etapas, fotos, clientes, invitaciones, links y los usuarios que solo pertenecían a él. Queda una línea en la auditoría de la plataforma. No se puede deshacer |

Por qué los 30 días: un borrado inmediato por error (taller equivocado, o pedido de alguien que no era el dueño) no tiene arreglo. Los SaaS reales siempre dejan un periodo de gracia.

Los respaldos diarios guardan copias hasta por 4 semanas. La política de privacidad debe decirlo (Lucero).

### Por qué todo en el VPS

Es la arquitectura que la app ya tiene hoy en Docker Compose: Caddy sirve la web y la API desde un mismo origen. Solo hay que llevarla al servidor y ponerle el dominio.

- **La web no cambia nada:** misma cookie de sesión y mismo CSRF, y no necesita CORS.
- **Sin límites de uso comercial**, a diferencia del plan gratis de Vercel.
- **Un solo lugar** para desplegar, revisar logs y respaldar.
- **El costo:** si el VPS cae, cae todo. Por eso los respaldos fuera del VPS y el monitoreo (Stephano) no son opcionales.

### "La web y la app conectadas a la misma base de datos"

La web y la app **no** se conectan a PostgreSQL. Las dos hablan con la misma API del VPS, y esa API es lo único que entra a la base. Así:

- un taller ve los mismos datos en la web y en el celular, al instante;
- las contraseñas de la base nunca viajan dentro de la app (una APK se puede descompilar);
- el aislamiento entre talleres (RLS) sigue en un solo lugar.

### Por qué la app usa token y no cookie

Dentro de Capacitor, la app corre en `https://localhost` (Android) y llama a `https://tallerflow.pe/api`. Para el navegador interno son sitios distintos, así que las cookies no son confiables. Por eso la API necesita CORS **solo** para la app móvil.

La app recibe el mismo token de sesión opaco que hoy va en la cookie, lo guarda cifrado en el Keystore de Android y lo manda en `Authorization: Bearer`. Como no viaja automáticamente con cada request, no necesita CSRF.

No es un JWT: sigue siendo una sesión en la base que se puede revocar, por ejemplo al suspender el taller.

### Celular: Android primero

En Gamarra domina Android. Publicar en Google Play cuesta un pago único (alrededor de 25 USD). iOS necesita una Mac y una cuenta de Apple Developer con pago anual (alrededor de 99 USD), así que queda para el Sprint 4.

Para el piloto basta un APK firmado que se instala directo, o la prueba interna de Play Console.

## 3. Antes del día 1 (José, este fin de semana o el lunes 19 temprano)

1. **Comprar el VPS.** Un solo servidor corre PostgreSQL, el backend, nginx y Caddy.
   - Tamaño: 2 vCPU, 4 GB de RAM, 40 GB o más de disco SSD, Ubuntu LTS. Con 2 GB, PostgreSQL y el build de Docker se quedan cortos.
   - Proveedor: Hetzner Cloud es el más barato por lo que da. DigitalOcean y Vultr cuestan más, pero son más simples de pagar con tarjeta peruana. Comparar el precio mensual vigente antes de elegir.
   - Región: la más cercana a Lima que ofrezca el proveedor (EE. UU. Este, Miami o São Paulo). Para esta app, de 100 a 150 ms de latencia no se notan.
   - Activar los snapshots automáticos del proveedor si cuestan poco. Son una segunda red de seguridad, aparte del `pg_dump` diario.
2. **Comprar el dominio.** Comparar antes de pagar:

   | Opción | Dónde se compra | A favor | En contra |
   |---|---|---|---|
   | `tallerflow.pe` o `tallerflow.com.pe` | Un registrador acreditado por Punto.pe (NIC Perú) | Genera confianza local en Gamarra y queda bien en `soporte@tallerflow.pe` | Más caro que un `.com` y no se puede comprar en Cloudflare |
   | `tallerflow.com` | Cloudflare Registrar | Precio de costo, privacidad WHOIS gratis y el DNS queda listo en el mismo lugar | Menos "peruano" |

   - Revisar disponibilidad y precio del primer año **y de la renovación** antes de pagar.
   - Activar la renovación automática y el bloqueo de transferencia. Si el dominio vence, se caen a la vez la web, la app, la consola, los links de los clientes y el correo.
   - Registrarlo a nombre del proyecto con un correo del equipo, no con uno personal.
3. **Anexar el dominio al VPS:**
   1. Crear una cuenta gratis en Cloudflare y agregar el dominio.
   2. Si se compró fuera de Cloudflare, cambiar en el registrador los nameservers por los dos que indica Cloudflare. Puede tardar unas horas en propagarse.
   3. Crear los registros DNS:

      | Nombre | Tipo | Apunta a | Proxy |
      |---|---|---|---|
      | `@` | A | IP del VPS | Activado (naranja) |
      | `www` | CNAME | `@` (y una regla que redirige a `https://<dominio>`) | Activado |
      | `admin` | A | IP del VPS | Activado (naranja) |
      | SPF, DKIM, DMARC | TXT/CNAME | Los que indique el proveedor de correo (Lucero, entregable 10) | Solo DNS |

   4. En Cloudflare, poner SSL/TLS en modo **Full (strict)**. Caddy obtiene su certificado solo en el VPS. Con el modo "Flexible", el tramo entre Cloudflare y el VPS viajaría sin cifrar.
   5. En Cloudflare Zero Trust (gratis para equipos pequeños), crear una aplicación **Access** para `admin.<dominio>` que solo deje pasar los correos de los dueños de TallerFlow, con un código enviado por correo. Así, ni la pantalla de login de la consola es visible para el resto de internet.
   6. En el VPS, cambiar las direcciones del sitio en `infra/caddy/Caddyfile` al dominio, y poner `TF_ALLOWED_ORIGIN`, `TF_PUBLIC_BASE_URL` y `TF_CONSOLE_ORIGIN` en el `.env` (sección 5.3).
   7. Levantar `compose.production.yaml` y comprobar `https://<dominio>/health/ready`.
4. **Crear las cuentas compartidas:** el proveedor de correo (Resend o Brevo; ambos tienen un plan gratis suficiente para el piloto) y Google Play Console, si se decide publicar ahí.
5. **Dar accesos:**
   - Stephano: Cloudflare y una llave SSH propia en el VPS el día 2.
   - Lucero: Cloudflare (solo DNS) y el proveedor de correo el día 2.
6. Crear un issue en GitHub por cada entregable de este documento y asignarlo.

Mientras algo de esto no esté listo, nadie espera:

- Mailpit reemplaza el correo.
- El Compose local reemplaza el VPS.
- `http://admin.localhost:8080` reemplaza el subdominio de la consola; Chrome y Edge resuelven `*.localhost` solos.

## 4. Reglas para no pisarnos

### Ramas

```powershell
git switch main
git pull --ff-only
git switch -c feat/s3-<persona>-<area>
```

| Persona | Rama |
|---|---|
| José | `feat/s3-jose-accounts-deploy` |
| Lucero | `feat/s3-lucero-mail-brand` |
| Michael | `feat/s3-michael-console-orders` |
| Stephano | `feat/s3-stephano-mobile-ops` |

- Nadie trabaja directo en `main`. Un PR por entregable, no uno gigante al final.
- Actualizar la rama con `main` todos los días (`git pull --rebase origin main`).
- Cada persona solo edita los archivos de su sección. Si necesitas tocar uno ajeno, avisa en el grupo y acuerden quién lo hace.
- Nunca subir contraseñas, llaves de API, `.env`, secretos de 2FA, keystores de Android ni `google-services.json` al repo, ni pegarlos en el chat. Los secretos viven en el VPS y en los secretos de GitHub Actions.

### Archivos compartidos

| Archivo | Regla |
|---|---|
| `backend/cmd/api/app.go` | Cada uno agrega solo las líneas de su módulo; José resuelve conflictos al fusionar |
| `backend/platform/config/config.go` | Cada uno agrega sus variables `TF_*`, siempre con valor por defecto para desarrollo |
| `frontend/src/app/router.ts` | Cada uno agrega sus rutas en un bloque `// Sprint 3 - <persona>` |
| `frontend/package.json` | Cada uno instala sus dependencias; el conflicto en `package-lock.json` se resuelve con `npm install` |
| `docs/contracts/openapi.yaml` | Cada uno edita solo sus rutas, y el contrato se actualiza **antes** que el código |
| `infra/caddy/Caddyfile` | José; Michael le pasa el bloque de `admin.` |
| `compose*.yaml`, `.env.example`, `infra/postgres/init-roles.sh` | Stephano; los demás le pasan la variable nueva por el grupo |
| `frontend/src/styles/` | Solo Lucero |

### Migraciones Flyway: números reservados

| Número | Dueño | Contenido |
|---|---|---|
| `V11` | José | Verificación de correo y registro abierto |
| `V12` | Michael | `completed_by` en etapas, una sola estación en proceso y campos de cancelación |
| `V13` | José | Tipo de sesión (`WEB` / `MOBILE`) en `user_sessions` |
| `V14` | Michael | Estado del taller, tablas de la consola y sus funciones SQL |
| `V15+` | Pedir a José | Cualquier otra necesidad |

Nunca editar una migración ya fusionada. Si tu base local se desordena porque se fusionaron en otro orden, recréala con `docker compose ... down --volumes`. Hay que recrearla igual el día 2, cuando Stephano agrega el rol `tallerflow_console`, porque `init-roles.sh` solo corre con un volumen vacío.

## 5. Contratos congelados el día 1

Con estos contratos cada uno avanza sin esperar al otro. Para cambiarlos, se acuerda en la reunión diaria.

### 5.1 Envío de correo (Lucero lo implementa; José, Lucero y Michael lo usan)

```go
// backend/platform/mail
package mail

type Message struct {
	To      string
	Subject string
	Text    string // siempre presente
	HTML    string // opcional
}

type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// Render arma Subject, Text y HTML desde una plantilla con marca.
func Render(template string, data any) (Message, error)
```

El día 1, Lucero sube un PR pequeño solo con esta interfaz y un `mail.Fake` en memoria para pruebas. Mientras no se fusione, José y Michael definen la misma interfaz dentro de su paquete: en Go, cualquier tipo con el método `Send` la cumple.

| Plantilla | Variables | La envía |
|---|---|---|
| `verify-email` | `.Name`, `.WorkshopName`, `.URL`, `.ExpiresIn` | José |
| `team-invitation` | `.WorkshopName`, `.InviterName`, `.Role`, `.URL`, `.ExpiresIn` | Lucero |
| `password-reset` | `.URL`, `.ExpiresIn` | Lucero |
| `workshop-suspended` | `.WorkshopName`, `.Reason`, `.SupportEmail` | Michael |
| `workshop-deletion-scheduled` | `.WorkshopName`, `.DeletionDate`, `.SupportEmail` | Michael |

Hasta que llegue el HTML se manda solo el texto plano. Cuando llegue, nadie cambia código.

### 5.2 Sesión de la app móvil (José lo implementa; Stephano lo consume)

| Paso | Contrato |
|---|---|
| Iniciar sesión | `POST /api/v1/auth/login` con `"client": "mobile"` en el cuerpo. Responde lo mismo que hoy más `session_token` y `expires_at`, y **no** pone cookie |
| Cada request | Header `Authorization: Bearer <session_token>`, sin cookie ni CSRF |
| Duración | 30 días, renovándose con el uso. Se revoca al cerrar sesión, al cambiar la contraseña, al desactivar al miembro o al suspender el taller |
| Cambio obligatorio de contraseña | `POST /api/v1/auth/change-password` con bearer responde un `session_token` nuevo |
| Cerrar sesión | `POST /api/v1/auth/logout` con bearer revoca ese token |
| Errores | `401` = token vencido o revocado: la app borra el token y va al login |

Request con cookie **y** bearer a la vez: responde `400`. Hasta que esto esté en `main`, Stephano prueba contra un mock en las pruebas de Vitest.

### 5.3 Variables de entorno nuevas

| Variable | Dueño | Desarrollo | Producción |
|---|---|---|---|
| `TF_ALLOWED_ORIGIN` (existe) | José | `http://localhost:8080` | `https://tallerflow.pe` |
| `TF_MOBILE_ORIGINS` | José | `http://localhost,https://localhost,capacitor://localhost` | `https://localhost,capacitor://localhost` |
| `TF_PUBLIC_BASE_URL` | José | `http://localhost:8080` | `https://tallerflow.pe` (base de los links de los correos) |
| `TF_SIGNUP_ENABLED` | José | `true` | `true` (en `false` cierra el registro si hay abuso) |
| `TF_CONSOLE_ORIGIN` | Michael | `http://admin.localhost:8080` | `https://admin.tallerflow.pe` |
| `TF_CONSOLE_DATABASE_URL` | Michael | Conexión con el rol `tallerflow_console` | Igual, con su contraseña de producción |
| `TF_CONSOLE_TOTP_KEY` | Michael | Valor fijo de desarrollo | 32 bytes aleatorios; cifra los secretos 2FA en la base |
| `DB_CONSOLE_PASSWORD` | Stephano | `local-console-change-me` | Aleatoria |
| `VITE_API_BASE_URL` | Stephano | vacío (rutas relativas, como hoy) | Web: vacío (mismo dominio). Android: `https://tallerflow.pe` |

Stephano agrega todas a `.env.example`, los compose e `init-roles.sh` (rol `tallerflow_console` con `NOBYPASSRLS`) el día 2. `VITE_API_BASE_URL` solo se llena en el build de Android.

### 5.4 Rutas nuevas del frontend

| Ruta | Dueño |
|---|---|
| `/` (landing para quien no tiene sesión) | Lucero |
| `/registro` | José |
| `/verificar-correo?token=...` | José |
| `/terminos`, `/privacidad` | Lucero |
| `/consola/*` (solo en el host `admin.`) | Michael |

El registro enlaza a `/terminos` y `/privacidad` desde el día 1, aunque las páginas aparezcan el día 5. **La app Android abre en `/login`, no en la landing**, y no incluye la consola.

### 5.5 Íconos de la app (Lucero los entrega; Stephano los usa)

Lucero deja estos archivos en `frontend/resources/` a más tardar el día 5:

- `icon.png` (1024×1024);
- `icon-foreground.png` e `icon-background.png` (ícono adaptativo de Android);
- `splash.png` (2732×2732, logo centrado sobre denim `#16294A`).

Stephano los convierte con `@capacitor/assets`. Hasta entonces usa un ícono provisional.

### 5.6 Cámara y escáner (Stephano lo implementa; Michael lo usa)

```ts
// frontend/src/native/camera.ts
// Abre la cámara nativa en Android o el selector de archivos en la web.
// Devuelve la foto ya comprimida (lado mayor 1600 px, JPEG 0.8), o null si se canceló.
export async function capturePhoto(): Promise<Blob | null>

// frontend/src/native/scanner.ts
export function canScanNatively(): boolean
// Devuelve el texto del código de la ficha, o null si se canceló o no hay permiso.
export async function scanOrderCode(): Promise<string | null>
```

Stephano lo entrega el día 4. Hasta entonces, Michael sigue con el código actual de fotos y escáner. El día 8 cambia esas llamadas por estas, que es un cambio de pocas líneas.

### 5.7 Estado del taller (Michael lo implementa; José lo muestra en el login)

- `workshops.status`: `ACTIVE`, `SUSPENDED` o `PENDING_DELETION`.
- Michael agrega el filtro en `workshops.Service.Resolve`: si el taller no está `ACTIVE`, devuelve `ErrWorkshopSuspended`. Eso corta tanto la sesión web como la móvil.
- José mapea ese error a `403` con código `WORKSHOP_SUSPENDED`. `LoginView` muestra: "Tu taller está suspendido. Escríbenos a `soporte@tallerflow.pe`".
- El link de seguimiento de un taller suspendido responde lo mismo que un link inválido (Michael).

## 6. José — Registro de talleres, VPS y sesión móvil

**Objetivo:** cualquier taller se registra solo; toda la app corre en el VPS con el dominio propio, y la API acepta tanto a la web como a la app Android.

### Cómo vive el registro un taller nuevo

1. Entra a `tallerflow.pe`, toca **"Crear mi taller"** y llega a `/registro`.
2. Escribe el nombre del taller, su nombre, su correo y una contraseña, y acepta términos y privacidad.
3. Ve "Te enviamos un correo a lucho@…", con un botón para reenviarlo.
4. Abre el correo con la marca de TallerFlow, toca "Verificar mi correo" y entra directo a su panel.
5. El panel vacío le muestra **"Primeros pasos"** (Lucero): crear su primer cliente, crear su primera orden, invitar a su equipo y descargar la app.
6. El taller aparece en la consola de plataforma como `ACTIVE`, con la fecha de alta.

**Archivos propios:**

```text
backend/internal/onboarding/
backend/internal/auth/
backend/platform/security/          (CORS y origen)
backend/platform/httpx/
database/migrations/V11__*.sql, V13__*.sql
frontend/src/modules/onboarding/    (pasa a /registro)
frontend/src/modules/auth/LoginView.vue
infra/caddy/Caddyfile
scripts/verify-sprint3.*
docs/runbooks/vps.md
```

**Entregables:**

1. **Registro abierto (`/registro`).** Crea el usuario, el taller (`ACTIVE`) y la membresía OWNER en una sola transacción.
   - El candado de un solo taller (`bootstrap_state`) deja de aplicarse al registro web. El CLI `cmd/bootstrap` queda para casos de soporte.
   - Se mantiene un taller por usuario (`workshop_members_one_workshop_per_user_idx`); varios talleres por persona quedan para otro sprint.
   - El registro exige aceptar términos y privacidad, y guarda la fecha de aceptación.
   - `LoginView` agrega el link "¿Tu taller aún no tiene cuenta? Créala aquí".
2. **Verificación de correo (`V11`).**
   - Agregar `users.email_verified_at` y la tabla `email_verification_tokens`. El token se guarda solo como hash SHA-256, vence en 24 horas y se usa una vez.
   - Los usuarios que ya existen quedan verificados en la misma migración.
   - `POST /api/v1/auth/email-verifications/consume` verifica la cuenta e inicia sesión.
   - `POST /api/v1/auth/email-verifications/resend` siempre responde igual, exista o no el correo.
   - Login sin verificar: `403` con código `EMAIL_NOT_VERIFIED`. `LoginView` muestra "Revisa tu correo" y un botón para reenviar.
3. **Abuso.** Rate limit por IP en registro y reenvío. El mensaje es el mismo exista o no el correo, para no revelar quién está registrado. `TF_SIGNUP_ENABLED=false` cierra el registro sin desplegar código.
4. **CORS solo para la app móvil.**
   - La web sigue en el mismo dominio que la API, así que su flujo con cookie y CSRF no cambia.
   - Middleware CORS con lista blanca exacta `TF_MOBILE_ORIGINS`, sin credenciales.
   - La cookie se acepta solo con el origen `TF_ALLOWED_ORIGIN`. El bearer se acepta desde los orígenes móviles y no exige CSRF.
5. **Sesión móvil (`V13`).** Implementar el contrato 5.2. Las sesiones `MOBILE` aparecen como tales en la auditoría.
6. **Taller suspendido en el login** (contrato 5.7).
7. **VPS en producción.**
   - Ubuntu con solo acceso por llave SSH (sin contraseña ni root) y actualizaciones automáticas.
   - Firewall: 22 abierto solo para el equipo; 80 y 443 abiertos solo para los rangos de IP de Cloudflare.
   - Docker con rotación de logs.
   - `compose.production.yaml` levanta PostgreSQL, Flyway, el backend, el frontend (nginx) y Caddy, igual que el Compose local.
   - Caddy sirve `tallerflow.pe` y `admin.tallerflow.pe`. Las rutas `/api/v1/console/*` solo responden en el host `admin.`, y en el host principal devuelven `404`. Cloudflare queda en modo Full (strict).
   - Caddy confía en `Cf-Connecting-IP` únicamente desde los rangos de Cloudflare, para que el rate limit vea la IP real.
   - Anexar el dominio siguiendo el paso 3 de la sección 3.
   - PostgreSQL sin puerto público. Secretos solo en el `.env` del servidor, con permisos `600`.
   - Documentar todo en `docs/runbooks/vps.md`.
8. **Integración (días 9 y 10).** Fusionar en el orden de la sección 10, crear `scripts/verify-sprint3.ps1` y `.sh` partiendo del de Sprint 2, y dirigir la demo.

**Pruebas mínimas:**

- Dos talleres registrados no ven datos uno del otro (integración con RLS).
- Un token de verificación vencido, ya usado o adulterado responde error.
- Un origen que no está en la lista recibe CORS denegado.
- Un bearer revocado responde `401`.
- `/api/v1/console/*` en el host principal responde `404`.
- E2E `signup-flow.spec.ts`: registro → correo en Mailpit → verificar → panel.

**PR sugeridos:**

- `feat(signup): open multi-workshop registration`
- `feat(auth): verify email before first login`
- `feat(auth): bearer sessions for the mobile app`
- `feat(security): CORS for the mobile app`
- `docs(runbook): production VPS`

**Plan B por si se atrasa:** José tiene la mayor carga. Si el día 6 el VPS no está listo, Stephano toma el entregable 7 con el runbook, y José se queda con el código.

**José no modifica:** `backend/platform/mail/`, estilos globales, órdenes, la consola ni el proyecto Android.

## 7. Michael — Consola de plataforma y órdenes sin miedo a equivocarse

**Objetivo:** que los dueños de TallerFlow administren los talleres como en un SaaS real, que un error de avance se deshaga en segundos y que la base nunca quede en un estado imposible.

**Archivos propios:**

```text
backend/internal/console/            (nuevo)
backend/cmd/consoleadmin/            (nuevo: CLI para crear administradores de plataforma)
backend/internal/workshops/          (filtro de estado, contrato 5.7)
backend/internal/orders/
database/migrations/V12__*.sql, V14__*.sql
frontend/src/modules/console/        (nuevo)
frontend/src/modules/orders/
frontend/src/modules/tracking/
frontend/e2e/operations-ui.spec.ts, customer-tracking.spec.ts, console.spec.ts
```

### 7.1 Primero: el error de dos estaciones en proceso (días 1 y 2)

La orden OT-000001 tiene Costura y Acabado "en proceso" a la vez desde el 30/09, y el cliente lo ve así en su link.

1. Encontrar qué camino del código lo permitió. El sospechoso principal es "Corregir avance" (`ReopenStage`). Escribir una prueba que lo reproduzca **antes** de arreglarlo.
2. Arreglar el código.
3. En `V12`, reparar los datos existentes y agregar un índice para que la base no lo vuelva a permitir:

```sql
CREATE UNIQUE INDEX order_stages_one_in_progress_idx
    ON order_stages (order_id) WHERE status = 'IN_PROGRESS';
```

Este índice también protege el deshacer de la sección siguiente: aunque haya un error en el código o dos personas toquen la misma orden a la vez, la base rechaza el estado imposible.

### 7.2 Deshacer un avance por error

**El problema real.** El operario escanea la ficha equivocada o toca "Pasar a…" en la tarjeta de al lado. Hoy solo un OWNER/ADMIN puede corregirlo, y tiene que escribir un motivo. En la práctica el operario no lo avisa, el error se queda y el cliente ve una estación que no corresponde.

**La solución: dos niveles.**

| Nivel | Quién | Cuándo | Cómo |
|---|---|---|---|
| **Deshacer** | La misma persona que avanzó, de cualquier rol | Hasta 5 minutos después del avance | Un toque, sin motivo |
| **Corregir avance** (existe) | OWNER o ADMIN | Cuando sea | Con motivo de 5 a 300 caracteres |

**Cómo se ve:**

1. Al confirmar un avance aparece un aviso abajo durante 10 segundos: **"OT-000003 pasó a Costura · Deshacer"**. El botón tiene al menos 44 px y funciona con teclado.
2. Si el aviso se cerró, el detalle de la orden muestra **"Deshacer avance"** mientras dure la ventana de 5 minutos, con el tiempo que queda.
3. Pasados los 5 minutos, el operario ve: "Ya no puedes deshacerlo. Pide a un administrador que use Corregir avance". El administrador sigue teniendo su botón de siempre.
4. Después de deshacer, la tarjeta vuelve a su estación con la costura un punto atrás, y el link del cliente muestra el estado corregido. El error queda en la auditoría interna, pero el cliente no lo ve.

**Reglas en el backend (`V12` + endpoint):**

- Agregar `order_stages.completed_by` (usuario que completó la etapa).
- `POST /api/v1/orders/:id/stages/:position/undo`, con CSRF o bearer. Se permite solo si:
  - quien lo pide es `completed_by`;
  - han pasado 5 minutos o menos desde `completed_at` (constante en el backend; el reloj del celular no cuenta);
  - es la última etapa completada;
  - la siguiente etapa sigue en proceso, sin fotos ni avances.
- Responde `409` con un código claro: `UNDO_WINDOW_EXPIRED`, `STAGE_NOT_LATEST` o `NEXT_STAGE_HAS_EVIDENCE`. El frontend muestra un mensaje distinto para cada uno.
- Usa `orders.version` para que dos acciones a la vez no se pisen.
- Registra `ORDER_STAGE_UNDONE` en `audit_events`, con quién lo hizo y desde qué cliente (web o móvil).
- **Completar Entrega por error también se deshace:** la orden vuelve de `COMPLETED` a `ACTIVE` y reaparece en el tablero. Hoy "Corregir avance" no cubre ese caso, porque exige una etapa siguiente en proceso. Hay que extenderlo también para OWNER/ADMIN.

**Prevenir antes que deshacer:** el diálogo de confirmación muestra el código de la orden en grande (mono, como en la ficha), además de la estación actual y la siguiente, para detectar que se escaneó la ficha equivocada antes de confirmar.

### 7.3 Cancelar orden

- `POST /api/v1/orders/:id/cancel`: motivo de 5 a 300 caracteres, solo OWNER y ADMIN.
- La orden sale del tablero y el cliente ve "Cancelada" (esa vista ya existe). Los campos de cancelación van en `V12`.
- Botón "Cancelar orden" en el detalle, con diálogo de confirmación.
- Editar los datos de una orden pasa al Sprint 4. Mientras tanto, si una orden se cargó mal, se cancela y se crea de nuevo.

### 7.4 Consola de plataforma

**Para quién:** los dueños de TallerFlow. No es un rol de taller: es otra cuenta, en otro subdominio, con su propio login.

**Pantallas** (`frontend/src/modules/console/`, rutas `/consola/*`):

1. **Resumen:**
   - talleres por estado;
   - talleres nuevos en los últimos 7 y 30 días;
   - talleres activos en los últimos 7 días (con algún inicio de sesión o avance);
   - dueños que no verificaron su correo después de 48 horas;
   - órdenes creadas en los últimos 30 días, sumando todos los talleres.
2. **Talleres:** tabla con búsqueda por nombre o correo del dueño. Columnas: nombre, correo del dueño, fecha de alta, última actividad, miembros, órdenes activas, espacio que ocupan sus fotos y estado.
3. **Detalle de un taller:** los mismos datos y las acciones:
   - **Suspender** (motivo obligatorio) y **Reactivar**;
   - **Reenviar verificación** al dueño;
   - **Programar eliminación** (motivo obligatorio) y **Cancelar eliminación**;
   - **Eliminar definitivamente**, que solo se habilita pasados los 30 días y pide escribir el nombre exacto del taller y un código 2FA nuevo.
4. **Auditoría:** quién hizo qué, a qué taller, cuándo, con qué motivo y desde qué IP.

Cada acción muestra un diálogo de confirmación que explica la consecuencia. Por ejemplo: "Las 4 personas de Confecciones Gamarra perderán el acceso de inmediato".

Arriba de toda la consola va una barra grafito fija que dice **"Consola de plataforma"**, para que nunca se confunda con la vista de un taller.

**Login y seguridad:**

- Cuentas propias en la tabla `console_admins`, separadas de los usuarios de los talleres. **No existe registro web**: se crean solo por terminal en el VPS.

  ```powershell
  docker compose exec backend /tallerflow-consoleadmin create --email jose@... --password-stdin
  ```

  El comando muestra una sola vez el código QR para la app de autenticación (Google Authenticator, Microsoft Authenticator o similar).
- El login pide correo, contraseña (Argon2id, el mismo hasher de `auth`) y el código de 6 dígitos de la app (TOTP). El secreto TOTP se guarda cifrado con `TF_CONSOLE_TOTP_KEY`.
- La sesión dura 8 horas como máximo y se cierra tras 30 minutos sin uso. La cookie es `__Host-tallerflow_console`, solo del host `admin.`, así que nunca se mezcla con la sesión de un taller.
- Origen comprobado contra `TF_CONSOLE_ORIGIN`, CSRF y rate limit, igual que la app.
- Antes del login de la consola está Cloudflare Access (sección 3, paso 3.5): son dos cerraduras.

**Base de datos (`V14`):**

- `workshops`:
  - `status` (`ACTIVE` por defecto para los talleres existentes);
  - `status_reason`;
  - `status_changed_at`;
  - `deletion_scheduled_for`.
- Tablas nuevas:
  - `console_admins` y `console_sessions`;
  - `console_audit_events`, de solo inserción: el rol no tiene permiso de `UPDATE` ni `DELETE`, y se conserva aunque el taller se elimine.
- El rol `tallerflow_console` **no** recibe permisos sobre las tablas de los talleres. Solo puede ejecutar funciones `SECURITY DEFINER`, cada una con `SET search_path` fijo:
  - `console_list_workshops(search, limit, offset)` y `console_workshop_detail(id)`: solo metadatos y conteos;
  - `console_set_workshop_status(id, status, reason, actor_id)`: además revoca todas las sesiones web y móviles de los miembros;
  - `console_purge_workshop(id, confirm_name, actor_id)`: verifica que hayan pasado los 30 días y que el nombre coincida. Borra en cascada el taller y los usuarios que solo pertenecían a él, y deja la línea de auditoría.
- La API de la consola usa su propia conexión (`TF_CONSOLE_DATABASE_URL`), igual que hoy se usa una conexión separada para el bootstrap.

**Correos al dueño:** al suspender, se envía `workshop-suspended`; al programar la eliminación, `workshop-deletion-scheduled` (contrato 5.1).

**Plan B por si se atrasa:** si el día 7 el 2FA no está listo, se fusiona la consola con correo y contraseña. Cloudflare Access hace de segundo factor hasta que llegue el TOTP, que no puede faltar al cierre del sprint.

### 7.5 Vistas de órdenes según `DESIGN.md`

- Quitar los estilos viejos que quedaron tapados en `OrderCreateView` y `OrderDetailView`.
- Usar los botones unificados de Lucero cuando lleguen (día 7).
- Usar `capturePhoto` y `scanOrderCode` de Stephano (contrato 5.6) desde el día 8.

**Pruebas mínimas:**

- La base rechaza dos estaciones en proceso.
- Deshacer después de 5 minutos, o hecho por otra persona, responde `409` o `403`.
- Deshacer Entrega devuelve la orden a `ACTIVE`.
- Un OPERATOR recibe `403` al cancelar.
- El rol `tallerflow_console` no puede hacer `SELECT` sobre `orders`, `clients` ni `stage_photos` (prueba SQL en `database/tests/`).
- Suspender un taller cierra las sesiones de sus miembros, y su próximo login recibe `WORKSHOP_SUSPENDED`.
- `console_purge_workshop` falla antes de los 30 días o con un nombre que no coincide.
- E2E a 360 px: avanzar → Deshacer desde el aviso → la tarjeta vuelve a su estación.
- E2E `console.spec.ts`: login con TOTP → suspender un taller → su dueño no puede entrar → reactivar.

**PR sugeridos:**

- `fix(orders): enforce a single in-progress stage`
- `feat(orders): undo a mistaken advance`
- `feat(orders): cancel orders`
- `feat(console): platform admin accounts with TOTP`
- `feat(console): workshop lifecycle and console screens`

**Michael no modifica:** autenticación de los talleres, correo, estilos globales, compose ni la configuración de Capacitor.

## 8. Lucero — Correo, recuperación, marca y páginas públicas

**Objetivo:** que todo lo que el cliente ve fuera del tablero (correos, landing, logo, ícono de la app, legales, primeros pasos) se vea profesional, y que los correos lleguen de verdad a la bandeja de entrada.

**Archivos propios:**

```text
backend/platform/mail/               (nuevo)
backend/internal/passwordreset/
backend/internal/team/               (envío de la invitación por correo)
frontend/src/modules/password-reset/
frontend/src/modules/team/TeamView.vue
frontend/src/modules/session/SessionHomeView.vue (solo "Primeros pasos")
frontend/src/modules/landing/        (nuevo)
frontend/src/modules/legal/          (nuevo)
frontend/src/components/
frontend/src/layouts/
frontend/src/styles/
frontend/public/
frontend/resources/                  (íconos de la app)
frontend/index.html
DESIGN.md                            (sección Logo)
```

**Entregables:**

1. **Día 1:** PR con la interfaz `mail.Sender` y `mail.Fake` (sección 5.1).
2. **Paquete `mail`.** Mover el envío SMTP de `passwordreset/delivery.go` a `platform/mail`, con las mismas validaciones de TLS que ya existen. `passwordreset` pasa a usarlo.
3. **Plantillas con marca.** Las cinco de la sección 5.1, en HTML y texto, embebidas con `go:embed`.
   - Llevan el logo, el denim `#1F3A68` y un botón grande.
   - Usan fuentes del sistema, porque los clientes de correo no cargan Google Fonts.
   - Revisarlas en Gmail (web y celular) y en Outlook.
4. **Invitación por correo.** `team.Invite` y "regenerar" envían `team-invitation`. El botón para copiar el link se queda como respaldo. `TeamView` muestra "Te enviamos la invitación a correo@…".
5. **Recuperación de punta a punta.** Probarla en Mailpit y luego con el proveedor real. Verificar que el link abre `https://tallerflow.pe/reset-password` (sale de `TF_PUBLIC_BASE_URL`). Pulir `ForgotPasswordView` y `ResetPasswordView` según `DESIGN.md`.
6. **Paquete de marca.** Partir del logo actual (etiqueta colgante con 7 puntadas, `BrandLogo.vue`) y entregar:
   - versiones horizontal, solo símbolo, monocromo y sobre denim, en SVG;
   - favicon `.ico`, PNG de 16, 32, 180, 192 y 512, ícono "maskable" y `manifest.webmanifest`;
   - imagen de 1200×630 para la vista previa del link por WhatsApp, con sus etiquetas `og:` en `index.html`;
   - íconos y splash de la app Android (sección 5.5), a más tardar el día 5.

   Si propone cambiar el logo, José lo aprueba antes del día 4 y se actualizan `BrandLogo.vue`, el favicon y la sección Logo de `DESIGN.md`.
7. **Landing en `/`.** Llevar al repo la landing que hoy vive fuera de él.
   - Quien no tiene sesión la ve; quien la tiene entra a `/app`.
   - Botones "Crear mi taller" (`/registro`), "Iniciar sesión" y "Descargar app Android" (link al APK o a Play, que pasa Stephano el día 9).
   - **No** muestra credenciales demo en producción.
8. **Primeros pasos.** En el panel de un taller nuevo, una lista que se marca sola con los datos del panel:
   - "Crea tu primer cliente";
   - "Crea tu primera orden";
   - "Invita a tu equipo";
   - "Descarga la app".

   Desaparece al completarla o al tocar "Ocultar". Un taller que entra por primera vez nunca debe ver una pantalla vacía sin saber qué hacer.
9. **Términos y privacidad** (`/terminos`, `/privacidad`) en lenguaje simple, alineados con la Ley 29733 de Protección de Datos Personales del Perú. Explican:
   - qué datos guardamos (taller, equipo, clientes, fotos) y para qué;
   - que el equipo de TallerFlow no mira el contenido de las órdenes;
   - dónde están los datos (el VPS y los proveedores: Cloudflare, el de VPS y el de correo);
   - cuánto tiempo se guardan;
   - cómo pedir que se borren: periodo de 30 días, y respaldos que se eliminan solos en 4 semanas;
   - en qué casos se puede suspender un taller.
10. **Correo real.**
    - Verificar el dominio en el proveedor con SPF, DKIM y DMARC; remitente `soporte@tallerflow.pe`.
    - Probar que llega a la bandeja de entrada de Gmail y Outlook, y no a spam.
    - Las credenciales SMTP se las pasa a José por un canal privado, para el `.env` del VPS. Nunca van al repo ni al chat del grupo.
11. **Botones unificados** en `main.css` (primario, secundario, peligro y fantasma), con altura mínima de 44 px. Se aplican en auth, `ClientsView` y `TeamView`; Michael y José los usan en sus vistas. Entrega el día 7.
12. **Si sobra tiempo:** modo oscuro con los tokens que ya define `DESIGN.md`.

**Pruebas mínimas:**

- Cada plantilla se renderiza sin variables vacías (prueba unitaria en Go).
- E2E de recuperación con Mailpit.
- La landing a 360 px, sin scroll horizontal.
- La vista previa de WhatsApp, probada con un link real.
- Prueba de entregabilidad (por ejemplo, mail-tester.com) con 9/10 o más.

**PR sugeridos:**

- `feat(mail): shared sender and branded templates`
- `feat(team): send invitations by email`
- `feat(brand): logo pack, app icons and share preview`
- `feat(public): landing, terms and privacy pages`
- `feat(home): first steps for new workshops`

**Lucero no modifica:** registro, verificación ni sesión (José), órdenes ni consola (Michael), compose, CI ni Capacitor (Stephano).

## 9. Stephano — App Android, seguridad web, CI/CD y respaldo

**Objetivo:** que la misma app Vue funcione en la web y en el celular con Capacitor, con cámara y escáner nativos; que cada cambio se pruebe y despliegue solo; y que no perdamos datos.

**Archivos propios:**

```text
frontend/capacitor.config.ts         (nuevo)
frontend/android/                    (nuevo, generado por Capacitor)
frontend/nginx.conf                  (cabeceras de seguridad)
frontend/src/modules/session/api.ts  (base de la API y token en móvil)
frontend/src/modules/session/session.store.ts (solo guardar y restaurar el token)
frontend/src/native/                 (nuevo: cámara, escáner, almacenamiento seguro)
compose*.yaml, .env.example, infra/postgres/init-roles.sh
.github/workflows/
backend/Dockerfile, frontend/Dockerfile
scripts/ (excepto verify-sprint3.* de José)
docs/runbooks/ (excepto vps.md)
```

**Entregables:**

1. **Día 2: variables nuevas y rol `tallerflow_console`** (sección 5.3), en `.env.example`, los compose e `init-roles.sh`.
2. **Cliente HTTP para web y móvil (`api.ts`).**
   - Usa `VITE_API_BASE_URL` como prefijo; vacío conserva las rutas relativas actuales, y el desarrollo local y los E2E no cambian.
   - En la web, `credentials: 'include'` y CSRF como hoy.
   - En Android, agrega `Authorization: Bearer` y no manda CSRF.
   - Un `401` en Android borra el token y va a `/login`. Un `403 WORKSHOP_SUSPENDED` muestra el mensaje de suspensión.
3. **Token guardado de forma segura.** Usar un plugin de almacenamiento cifrado sobre el Keystore de Android (por ejemplo `@aparajita/capacitor-secure-storage`; verificar que sea compatible con la versión de Capacitor). **Nunca** `localStorage` ni `@capacitor/preferences`, que guardan en texto plano. La web sigue sin guardar nada: usa la cookie.
4. **Cámara y escáner nativos** (contrato 5.6), entregados el día 4.
   - Fotos con `@capacitor/camera` en Android y el `<input type="file">` actual en la web. Siempre comprimidas antes de subir (lado mayor de 1600 px, JPEG 0.8): una foto pasa de 3–5 MB a 200–400 KB, la base crece menos y la subida funciona con datos móviles.
   - Escáner con `@capacitor-mlkit/barcode-scanning` en Android; en la web se queda `BarcodeDetector`.
   - Pedir el permiso de cámara justo antes de usarla, con un texto que explique para qué. Si se niega, devolver `null`, y la pantalla de Michael ofrece el ingreso manual.
5. **Cabeceras de seguridad de la web** en `frontend/nginx.conf`, que ya maneja la caché:
   - CSP que permita solo el propio dominio, Google Fonts e imágenes `data:` (las fotos de evidencia);
   - `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin` y `Permissions-Policy` con cámara solo para el propio sitio;
   - en el host `admin.`, además `X-Frame-Options: DENY` y `X-Robots-Tag: noindex`;
   - HSTS en Caddy (coordinar con José), una vez que el dominio funcione bien con HTTPS.

   Verificar en el navegador que la consola no muestre bloqueos de CSP en el panel, las órdenes, el escáner, el link de seguimiento y la consola de plataforma.
6. **App Android con Capacitor.**

   ```powershell
   npm --prefix frontend install @capacitor/core @capacitor/android @capacitor/app @capacitor/splash-screen @capacitor/camera @capacitor-mlkit/barcode-scanning
   npm --prefix frontend install -D @capacitor/cli @capacitor/assets
   npx --prefix frontend cap init TallerFlow pe.tallerflow.app --web-dir dist
   npx --prefix frontend cap add android
   npm --prefix frontend run build; npx --prefix frontend cap sync android
   ```

   - Abre en `/login`. El botón atrás de Android navega hacia atrás y cierra la app desde el panel. Las rutas `/consola/*` no existen en la app.
   - Splash e íconos de Lucero (sección 5.5).
   - Proyecto Android funcionando el **día 3**.
   - APK firmado. La keystore se guarda fuera del repo, en los secretos de GitHub y en una copia segura de José: **si se pierde, no se puede actualizar la app en Play**.
   - Distribución del piloto: APK directo o prueba interna de Play Console. Pasar el link a Lucero el día 9 para la landing.
   - Probar en al menos dos celulares reales: uno de gama baja con Android 10 u 11 y uno reciente.
7. **CI/CD.**
   - El CI corre en cada PR y `main` exige el CI verde y una revisión.
   - El workflow de despliegue es manual (`workflow_dispatch`): entra por SSH al VPS, hace `git pull`, reconstruye backend y frontend y levanta con `--wait`. Si `/health/ready` falla, vuelve al commit anterior.
   - Un job compila el APK de Android en cada tag `v*`.
8. **Copias de seguridad** (con la llave SSH que José da el día 2).
   - `pg_dump` diario fuera del VPS (por ejemplo, un bucket de Cloudflare R2), guardando 7 diarios y 4 semanales.
   - Probar una restauración completa en una base vacía y anotar cuánto tardó. Una copia que nunca se restauró no cuenta.
9. **Monitoreo.** UptimeRobot (o similar) sobre `https://tallerflow.pe/health/ready` y sobre la página de inicio, con aviso al grupo. Alerta si el disco del VPS pasa el 80%.
10. **Runbooks** en `docs/runbooks/`:
    - desplegar y volver a la versión anterior;
    - restaurar un respaldo;
    - publicar una versión de la app;
    - rotar secretos;
    - crear o quitar un administrador de la consola.

**Pruebas mínimas:**

- `verify-sprint2` sigue verde con `VITE_API_BASE_URL` vacío.
- Pruebas de Vitest del cliente HTTP, de `capturePhoto` y de `scanOrderCode`, en modo web y en modo nativo simulado.
- Desde una red externa, el puerto 5432 del VPS está cerrado.
- La restauración del respaldo funciona.

**PR sugeridos:**

- `feat(web): security headers and API base URL`
- `feat(native): camera and scanner with web fallback`
- `feat(mobile): Android app with Capacitor and secure session storage`
- `ci: build, deploy and Android release workflows`
- `feat(ops): backups, restore drill and uptime monitoring`

**Stephano no modifica:** la lógica de autenticación del backend, órdenes, consola, correo ni vistas.

## 10. Calendario

| Días | José | Lucero | Michael | Stephano |
|---|---|---|---|---|
| 0 | Compra VPS y dominio, crea cuentas, configura Cloudflare | — | — | — |
| 1–2 | OpenAPI de registro y sesión móvil, `V11` | PR de interfaz `mail`, mover SMTP, verificar dominio en el proveedor | Reproducir y arreglar las dos estaciones, `V12` | Variables, rol de consola, cliente HTTP con base URL |
| 3–5 | Registro, verificación, login sin verificar | Plantillas HTML, paquete de marca, íconos de la app | Deshacer avance (API + aviso + detalle), cancelar orden | Proyecto Android (día 3), cámara y escáner (día 4), cabeceras de seguridad |
| 6–8 | CORS, bearer, `V13`, taller suspendido, VPS en producción | Invitación por correo, landing, legales, primeros pasos, botones | `V14`, consola: login con TOTP, pantallas y ciclo de vida | Token seguro en Android, CI/CD, respaldos |
| 9 | Fusionar todo en orden | Correos reales en Gmail y Outlook | Usar cámara y escáner nativos, E2E | APK firmado, monitoreo, despliegue |
| 10 | Verificación final y demo | Corregir lo que salga | Corregir lo que salga | Ensayo de restauración, runbooks |

**Orden de fusión:** Stephano (variables, rol y cliente HTTP) → Lucero (`mail`) → Michael (órdenes) → José (cuentas, sesión móvil y CORS) → Michael (consola) → Stephano (Android contra la API real) → integración final de José.

**Reunión diaria de 15 minutos:** qué terminé, qué hago hoy, qué me bloquea. Un bloqueo de más de medio día se avisa apenas aparece.

## 11. Revisión de PR

| Autor | Revisa | Foco |
|---|---|---|
| José | Michael | Tokens, CORS, RLS al crear talleres, firewall del VPS |
| Lucero | José | Correos sin datos sensibles, entregabilidad, accesibilidad, `DESIGN.md` |
| Michael | José | Reglas de deshacer, funciones `SECURITY DEFINER`, permisos del rol de consola, 2FA |
| Stephano | José | Dónde se guarda el token, permisos de cámara, secretos fuera del repo, keystore |

Checklist del autor:

```markdown
- [ ] La rama parte de main actualizado.
- [ ] Solo toqué archivos de mi sección (o lo acordé en el grupo).
- [ ] Actualicé OpenAPI antes del código si cambió la API.
- [ ] Escribí la prueba que falla antes del arreglo.
- [ ] npm --prefix frontend test, npm --prefix frontend run build y go -C backend test ./... pasan.
- [ ] Probé a 360 px y con teclado si cambié pantallas; en celular real si toqué algo nativo.
- [ ] No hay secretos, .env, keystores, secretos 2FA ni credenciales en el diff.
- [ ] Las consultas a tablas de taller pasan por WithinTenant (o por una función de consola revisada).
```

## 12. Definición de terminado

**Talleres:**

- [ ] Dos talleres nuevos se registran en `https://tallerflow.pe` y ninguno ve datos del otro.
- [ ] El correo de verificación llega a la bandeja de entrada de Gmail y Outlook; sin verificar no se puede entrar.
- [ ] Un taller nuevo ve "Primeros pasos" al entrar por primera vez.
- [ ] El OWNER invita a un operario por correo y el operario entra con el link.
- [ ] Recuperar contraseña funciona con correo real.

**Celular:**

- [ ] La app Android inicia sesión, sobrevive a cerrarla y abrirla, toma fotos con la cámara nativa y escanea la ficha.
- [ ] Un cambio hecho en la app se ve en la web al recargar, y al revés.

**Órdenes:**

- [ ] Un avance por error se deshace en un toque dentro de 5 minutos; después, solo el administrador puede corregirlo.
- [ ] La base rechaza dos estaciones en proceso.
- [ ] Una orden se cancela; completar Entrega por error también se deshace.

**Consola de plataforma:**

- [ ] Solo se entra pasando Cloudflare Access, contraseña y código 2FA.
- [ ] Muestra la lista de talleres con sus métricas, sin ningún dato de órdenes ni clientes.
- [ ] Suspender un taller saca a todos sus miembros, en la web y en la app.
- [ ] Reactivarlo les devuelve el acceso con todos sus datos.
- [ ] Una eliminación programada se puede cancelar; la definitiva solo se permite pasados 30 días y queda en la auditoría.

**Publicación:**

- [ ] El link de seguimiento muestra la vista previa con el logo al compartirlo por WhatsApp.
- [ ] Landing, términos y privacidad están publicados, y el registro exige aceptarlos.
- [ ] PostgreSQL no es accesible desde internet; hay respaldo diario y una restauración probada.
- [ ] `scripts/verify-sprint3` pasa desde volúmenes vacíos y el CI está verde en `main`.

## 13. Demo de cierre

1. Stephano abre `https://tallerflow.pe` y muestra el monitoreo en verde.
2. Lucero recorre la landing y los términos en el celular.
3. José registra un taller nuevo con un correo real, lo verifica y muestra "Primeros pasos". Luego invita a un operario.
4. El operario acepta la invitación e instala la app Android desde el link de la landing.
5. En la app, el operario escanea la ficha, avanza la orden **por error** y la deshace desde el aviso.
6. Avanza la correcta y sube una foto con la cámara. José recarga la web y la orden aparece ya actualizada.
7. Michael entra a `admin.tallerflow.pe` con su código 2FA, muestra el taller recién creado en la lista y lo suspende. En el celular del operario, la app lo saca al login con el mensaje de suspensión. Luego lo reactiva y todo vuelve.
8. Lucero muestra "Olvidé mi contraseña" con el correo con marca.
9. Stephano restaura el respaldo de anoche en una base vacía.

## 14. Fuera de este sprint (Sprint 4)

Esto completa TallerFlow como SaaS comercial. Queda anotado para no perderlo:

| Tema | Por qué esperar |
|---|---|
| Planes y cobro | Prueba gratis de 30 días y luego un pago mensual con una pasarela que funcione en Perú (por ejemplo Culqi, Niubiz o Mercado Pago). Primero hay que validar en el piloto cuánto pagaría un taller. La suspensión de este sprint ya sirve para cortar por falta de pago |
| Límites por plan | Cantidad de miembros, órdenes al mes o espacio de fotos, según el plan. Depende de lo anterior |
| Exportar los datos del taller | Descargar órdenes y clientes en Excel o CSV antes de irse. Es una buena práctica y da confianza al vender |
| Acceso de soporte con permiso del dueño | El dueño del taller autoriza por un tiempo limitado que soporte vea su taller, y todo queda auditado |
| Aprobación manual de talleres nuevos | Útil si aparece spam en el registro. Mientras tanto, `TF_SIGNUP_ENABLED=false` y la suspensión cubren el caso |
| Editar los datos de una orden | En este sprint se resuelve cancelando y creando de nuevo |
| Fotos a Cloudflare R2 | Con la compresión de la sección 5.6, la base aguanta el piloto: unas 10 000 fotos ocupan alrededor de 3 GB |
| App iOS | Necesita una Mac y la cuenta de Apple Developer |
| Trabajar sin internet | Requiere una cola de avances pendientes y resolver conflictos; primero hay que medir cuánto falla la señal en los talleres |
| Notificaciones push | Necesita Firebase y una decisión sobre qué avisar |
| Varios talleres por persona | Cambia el índice de un taller por usuario y agrega un selector de taller |
| Página de estado pública | Para avisar a los talleres cuando hay una caída o un mantenimiento |
