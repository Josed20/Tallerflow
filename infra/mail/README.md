# Entrega de recuperación

Desarrollo usa Mailpit: `docker compose -f compose.mail.yaml up -d`. La bandeja se abre en `http://localhost:8025` y SMTP escucha en `localhost:1025`.

Producción configura el mismo adaptador mediante variables, sin guardar credenciales en el repositorio:

- `TF_SMTP_ADDRESS` (`host:puerto`)
- `TF_SMTP_USERNAME` y `TF_SMTP_PASSWORD`
- `TF_SMTP_FROM`
- `TF_SMTP_TLS=true` para STARTTLS
- `TF_PASSWORD_RESET_URL`, origen HTTPS público usado en el enlace

El token crudo solo existe en memoria durante la construcción y entrega del correo. PostgreSQL conserva únicamente SHA-256 y el backend no registra direcciones ni enlaces de recuperación.
