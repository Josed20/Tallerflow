# Runbook: servidor de producción (VPS)

TallerFlow corre completo en un solo VPS con Docker Compose: PostgreSQL, Flyway, el backend Go, nginx con la web y Caddy. El tráfico público entra por **Cloudflare Tunnel**: el contenedor `cloudflared` abre la conexión desde el servidor hacia Cloudflare, así que el VPS no publica ningún puerto web. Solo queda abierto SSH.

```text
Visitante ──▶ https://tallerflowpe.com (Cloudflare: DNS, HTTPS, protección)
                       │ túnel "tallerflow" (sale del VPS por el puerto 7844)
                       ▼
VPS ── cloudflared ──▶ Caddy (gateway:80) ──▶ /api, /health → backend
                                         └──▶ todo lo demás → nginx (web Vue)
       PostgreSQL: solo red interna de Docker, sin puerto publicado
```

**Cambio respecto al plan del Sprint 3:** el plan abría los puertos 80 y 443 a los rangos de Cloudflare y usaba Full (strict) con certificado en Caddy. Con el túnel no hace falta ninguna de las dos cosas: Cloudflare cifra el tramo público, el tramo VPS ↔ Cloudflare va dentro del túnel, y la IP del servidor no sirve para saltarse Cloudflare.

| Dato | Valor |
|---|---|
| Proveedor | Contabo, Cloud VPS 4 (4 vCPU, 8 GB, 100 GB SSD), x86_64 |
| Sistema | Ubuntu 24.04 LTS |
| Dominio | `tallerflowpe.com` (Cloudflare Registrar, renovación automática activada) |
| Túnel | `tallerflow`, id `fe78febd-fd73-45e1-bd16-c37c68745285` |
| Carpeta de la app | `/opt/tallerflow` |
| Secretos | `/opt/tallerflow/.env` y `/etc/tallerflow/tunnel-credentials.json`, nunca en el repo |

## 0. Antes de empezar

- La IP del VPS y la contraseña de `root` que llegan por correo de Contabo.
- Una llave SSH en tu PC. Si no tienes, en Git Bash: `ssh-keygen -t ed25519 -C "jose@tallerflow"`.
- El archivo de credenciales del túnel: `C:\Users\JOSE\.cloudflared\fe78febd-fd73-45e1-bd16-c37c68745285.json`.
- **Credenciales SMTP reales** (Resend o Brevo) con el dominio verificado. `compose.production.yaml` no arranca sin `TF_SMTP_HOST`, `TF_SMTP_USER`, `TF_SMTP_PASSWORD` y `TF_SMTP_FROM`.

## 1. Primer ingreso y endurecimiento

Desde Git Bash en tu PC:

```bash
ssh-copy-id root@<IP>        # pide la contraseña de Contabo una sola vez
ssh root@<IP>
```

En el servidor:

```bash
adduser --disabled-password --gecos "" deploy
usermod -aG sudo deploy
install -d -m 700 -o deploy -g deploy /home/deploy/.ssh
cp /root/.ssh/authorized_keys /home/deploy/.ssh/
chown deploy:deploy /home/deploy/.ssh/authorized_keys
passwd deploy                # contraseña para sudo; guárdala en el gestor del equipo

apt update && apt -y full-upgrade
apt -y install unattended-upgrades ufw git
dpkg-reconfigure -f noninteractive unattended-upgrades
```

Antes de cerrar la sesión de `root`, abre **otra** terminal y comprueba que `ssh deploy@<IP>` entra con la llave. Recién entonces bloquea las contraseñas y el ingreso de `root`:

```bash
cat > /etc/ssh/sshd_config.d/99-tallerflow.conf <<'EOF'
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
EOF
systemctl reload ssh

ufw default deny incoming
ufw default allow outgoing
ufw allow OpenSSH
ufw enable
```

Docker se salta `ufw` para los puertos que publica. Por eso producción no publica ninguno: `compose.production.yaml` quita el de PostgreSQL y `compose.tunnel.yaml` quita el de Caddy.

## 2. Docker

Como `deploy`:

```bash
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker deploy
sudo tee /etc/docker/daemon.json >/dev/null <<'EOF'
{ "log-driver": "local", "log-opts": { "max-size": "20m", "max-file": "5" } }
EOF
sudo systemctl restart docker
exit                          # vuelve a entrar para que tome el grupo docker
```

## 3. Código y secretos

```bash
sudo install -d -o deploy -g deploy /opt/tallerflow
git clone https://github.com/Josed20/Tallerflow.git /opt/tallerflow
cd /opt/tallerflow
```

Si el repositorio es privado, crea una *deploy key* de solo lectura: `ssh-keygen -t ed25519 -f ~/.ssh/tallerflow_deploy`, súbela en GitHub → Settings → Deploy keys y clona con `git@github.com:Josed20/Tallerflow.git`.

Genera las contraseñas en el servidor, no en el chat:

```bash
for v in POSTGRES_SUPERUSER_PASSWORD DB_OWNER_PASSWORD DB_MIGRATION_PASSWORD DB_APP_PASSWORD DB_BOOTSTRAP_PASSWORD DB_CONSOLE_PASSWORD TF_SESSION_PEPPER TF_CONSOLE_TOTP_KEY; do
  echo "$v=$(openssl rand -hex 32)"
done > .env.secrets
```

Crea `/opt/tallerflow/.env` con `nano .env`, copiando los valores de `.env.secrets`:

```dotenv
COMPOSE_PROJECT_NAME=tallerflow
POSTGRES_DB=tallerflow
POSTGRES_SUPERUSER=postgres
POSTGRES_SUPERUSER_PASSWORD=<generada>
DB_OWNER_PASSWORD=<generada>
DB_MIGRATION_PASSWORD=<generada>
DB_APP_PASSWORD=<generada>
DB_BOOTSTRAP_PASSWORD=<generada>
DB_CONSOLE_PASSWORD=<generada>
TF_DATABASE_URL=postgres://tallerflow_app:<DB_APP_PASSWORD>@postgres:5432/tallerflow?sslmode=disable
TF_BOOTSTRAP_DATABASE_URL=postgres://tallerflow_bootstrap:<DB_BOOTSTRAP_PASSWORD>@postgres:5432/tallerflow?sslmode=disable
TF_CONSOLE_DATABASE_URL=postgres://tallerflow_console:<DB_CONSOLE_PASSWORD>@postgres:5432/tallerflow?sslmode=disable
TF_SESSION_PEPPER=<generada>
TF_CONSOLE_TOTP_KEY=<generada>
TF_ALLOWED_ORIGIN=https://tallerflowpe.com
TF_PUBLIC_BASE_URL=https://tallerflowpe.com
TF_PASSWORD_RESET_BASE_URL=https://tallerflowpe.com
TF_CONSOLE_ORIGIN=https://admin.tallerflowpe.com
TF_MOBILE_ORIGINS=https://localhost,capacitor://localhost
TF_SIGNUP_ENABLED=true
TF_SMTP_HOST=<del proveedor>
TF_SMTP_PORT=587
TF_SMTP_USER=<del proveedor>
TF_SMTP_PASSWORD=<del proveedor>
TF_SMTP_FROM=soporte@tallerflowpe.com
TF_TUNNEL_CREDENTIALS_FILE=/etc/tallerflow/tunnel-credentials.json
```

```bash
chmod 600 .env
shred -u .env.secrets
```

Las contraseñas en hexadecimal no necesitan escaparse dentro de las URL. `TF_SESSION_PEPPER` no se cambia después: invalidaría todas las sesiones. `TF_CONSOLE_TOTP_KEY` tampoco: cifra los secretos 2FA de la consola guardados en la base.

`init-roles.sh` crea los roles de PostgreSQL, incluido `tallerflow_console`, **solo** la primera vez, con el volumen vacío. Cambiar después una contraseña `DB_*_PASSWORD` en el `.env` no cambia la del rol: hay que rotarla a propósito (runbook de rotación de secretos de Stephano).

## 4. Credenciales del túnel

Desde Git Bash en tu PC:

```bash
scp ~/.cloudflared/fe78febd-fd73-45e1-bd16-c37c68745285.json deploy@<IP>:/tmp/tunnel.json
```

En el servidor (el contenedor corre como el usuario `65532`):

```bash
sudo install -d -m 750 /etc/tallerflow
sudo install -m 400 -o 65532 -g 65532 /tmp/tunnel.json /etc/tallerflow/tunnel-credentials.json
rm /tmp/tunnel.json
```

## 5. Levantar

**Primero apaga el túnel de tu PC** (Ctrl+C en su ventana). Si los dos corren a la vez, Cloudflare reparte las visitas entre tu PC y el VPS.

```bash
cd /opt/tallerflow
docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml --profile app up -d --build --wait
docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml --profile app ps
curl -fsS https://tallerflowpe.com/health/ready
```

Desde tu PC, `cloudflared tunnel info tallerflow` debe mostrar un solo conector con arquitectura `linux_amd64`.

**Crear el taller de TallerFlow enseguida.** Mientras el registro abierto no esté en `main`, `/onboarding` deja crear **solo el primer** taller y su OWNER. Entra a `https://tallerflowpe.com/onboarding` apenas el sitio responda, para que nadie más lo ocupe.

## 6. Comprobaciones de seguridad

Desde tu PC (o cualquier red externa):

```bash
nc -zv -w 5 <IP> 5432   # debe fallar
nc -zv -w 5 <IP> 80     # debe fallar
nc -zv -w 5 <IP> 22     # debe conectar
```

En el servidor, `sudo ss -tlnp` no debe mostrar nada escuchando en `0.0.0.0` aparte de `sshd`.

## 7. Desplegar una versión nueva y volver atrás

```bash
cd /opt/tallerflow
git rev-parse --short HEAD > ~/previous-release   # anota la versión actual
git pull --ff-only
docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml --profile app up -d --build --wait
curl -fsS https://tallerflowpe.com/health/ready
```

Si la nueva versión falla:

```bash
git checkout "$(cat ~/previous-release)"
docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml --profile app up -d --build --wait
```

Volver atrás el código **no** deshace migraciones de Flyway. Si la versión nueva traía una migración que rompe la anterior, hay que restaurar el respaldo (runbook de Stephano).

## 8. Logs y diagnóstico

```bash
docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml --profile app logs --tail 200 backend
docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml --profile app logs --tail 100 cloudflared
df -h /                 # alerta si pasa el 80%
```

| Síntoma | Causa probable |
|---|---|
| Cloudflare muestra **Error 1033** | `cloudflared` no está conectado: revisa sus logs y que el archivo de credenciales exista con dueño `65532` |
| **Error 502** de Cloudflare | El túnel conecta pero Caddy o el backend no responden: `docker compose ... ps` y logs del backend |
| Iniciar sesión falla con error de origen | `TF_ALLOWED_ORIGIN` no es exactamente `https://tallerflowpe.com` |
| El backend no arranca | Falta alguna variable obligatoria: el mensaje `set a production ...` dice cuál |

## Pendiente

- `www.tallerflowpe.com`: redirección a `https://tallerflowpe.com` con una regla de Cloudflare.
- `admin.tallerflowpe.com` para la consola (Michael): una regla más en `infra/cloudflared/config.yml`, Cloudflare Access y el bloque de Caddy.
- Respaldos diarios fuera del VPS, restauración probada y monitoreo (Stephano).
- Despliegue manual desde GitHub Actions (Stephano).
