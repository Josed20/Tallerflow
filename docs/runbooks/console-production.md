# Consola de plataforma en producción

La consola usa dos barreras: Cloudflare Access en `admin.tallerflowpe.com` y el
login propio con contraseña y código TOTP de seis dígitos. El backend solo
acepta las rutas de consola cuando el `Host` coincide con `TF_CONSOLE_ORIGIN`.

## Habilitar el host

1. José crea la ruta DNS de `admin.tallerflowpe.com` hacia el túnel de
   `tallerflowpe.com` y una aplicación de Cloudflare Access para **todo** el
   host. Autoriza únicamente a los administradores de plataforma.
2. Desplegar la configuración de `infra/cloudflared/config.yml` y
   `infra/caddy/Caddyfile` de este repositorio. La ruta de Caddy para `admin.`
   deja pasar `/api/v1/console/*`, bloquea el resto de `/api/*` y sirve la
   interfaz web.
3. Comprobar en el VPS que `.env` tiene `TF_CONSOLE_ORIGIN` igual a
   `https://admin.tallerflowpe.com`, `TF_CONSOLE_DATABASE_URL` y una
   `TF_CONSOLE_TOTP_KEY` aleatoria de al menos 32 bytes. No cambiar la clave
   TOTP después de crear administradores: cifra sus secretos guardados.
4. Aplicar las migraciones y comprobar que el backend inicia sin errores.

## Crear el administrador

Hacerlo en el VPS, desde `/opt/tallerflow`, después de desplegar. Sustituir el
correo de ejemplo por el del administrador. La contraseña se lee sin mostrarla
ni guardarla en el historial del shell.

```bash
read -rsp 'Contraseña del administrador: ' TF_NEW_ADMIN_PASSWORD
printf '\n'
printf '%s\n' "$TF_NEW_ADMIN_PASSWORD" | docker compose \
  -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml \
  --profile app run --rm --no-deps -T \
  --entrypoint /tallerflow-consoleadmin backend \
  create --email admin@example.com --password-stdin
unset TF_NEW_ADMIN_PASSWORD
```

La herramienta imprime una URI `otpauth://` una sola vez. Agregarla a una
aplicación autenticadora del administrador y guardar el método de recuperación
en un lugar privado. No copiar la URI, la contraseña ni los códigos al chat,
al repositorio o a un ticket.

## Prueba de cierre

1. Desde una sesión sin autorización en Cloudflare Access, comprobar que el
   host `admin.` no deja ver la consola. Después, entrar con la identidad
   autorizada.
2. Abrir `https://admin.tallerflowpe.com/consola`. Una contraseña o código
   incorrectos deben fallar. Con los datos correctos debe aparecer la lista de
   talleres.
3. Elegir **solo un taller de prueba** con su dueño y operario preparados.
   Suspenderlo con un motivo. Confirmar que una sesión web y una sesión Android
   ya no pueden usar el taller.
4. Reactivar el mismo taller enseguida. Confirmar que el dueño y el operario
   recuperan el acceso y que ambos cambios figuran en la auditoría.

Si la prueba se interrumpe tras suspender, reactivar el taller antes de darla
por terminada. No probar suspensión con un taller real sin coordinar con su
dueño.
