# Infraestructura de Correo — TallerFlow

Este directorio documenta el servicio de entrega de correos electrónicos para el entorno local y de producción.

## Entorno Local (Desarrollo y Pruebas)

En desarrollo local, TallerFlow utiliza **Mailpit**, un servidor SMTP ligero que intercepta todos los correos salientes y ofrece una interfaz web de inspección:

- **SMTP:** `127.0.0.1:1025`
- **Web UI:** [http://localhost:8025](http://localhost:8025)

### Iniciar Mailpit

```powershell
docker compose -f compose.yaml -f compose.mail.yaml --profile app up -d --build
```

Esto levanta PostgreSQL, Flyway, Mailpit, backend, frontend y gateway. El backend
usa el hostname interno `mailpit:1025`, mientras la bandeja sigue disponible en
[http://localhost:8025](http://localhost:8025).

Para ejecutar únicamente Mailpit junto con un backend Go iniciado de forma nativa:

```powershell
docker compose -f compose.mail.yaml up -d mailpit
```

Todos los correos enviados para restablecimiento de contraseña (`/api/v1/auth/password-resets`) aparecerán en la bandeja de entrada web de Mailpit sin salir a internet.

## Entorno de Producción

En producción, la entrega se realiza mediante el adaptador SMTP estándar (`net/smtp` de Go), configurado mediante variables de entorno:

- `TF_SMTP_HOST`: Host del servidor SMTP (ej. `smtp.sendgrid.net`).
- `TF_SMTP_PORT`: Puerto SMTP (ej. `587` o `25`).
- `TF_SMTP_USER`: Usuario autenticado.
- `TF_SMTP_PASSWORD`: Contraseña o API key (inyectada por el orquestador).
- `TF_SMTP_FROM`: Dirección remitente (ej. `soporte@tallerflow.pe`).
- `TF_SMTP_REQUIRE_TLS`: Debe ser `true` en producción.
- `TF_PASSWORD_RESET_BASE_URL`: Origen HTTPS público usado para construir el enlace de recuperación.

El adaptador exige TLS para cualquier host SMTP que no sea loopback, incluso sin credenciales. En producción se debe indicar `RequireTLS: true`; exige STARTTLS con TLS 1.2 o superior, aplica un timeout de 10 segundos por defecto y no permite autenticación SMTP en claro. La única excepción es Mailpit local: `RequireTLS: false`, sin credenciales, en `127.0.0.1:1025`.

### Seguridad

- **Sin secretos en el repositorio:** Ninguna credencial SMTP se almacena en el control de versiones.
- **Sin tokens en logs:** El contenido de los correos y los tokens de recuperación temporales nunca se escriben en los logs del servidor.
