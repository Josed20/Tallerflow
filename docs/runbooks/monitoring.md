# Monitoreo de TallerFlow

La puesta en marcha requiere el dominio HTTPS y acceso al VPS de Jose.

1. En el servicio de monitoreo externo, crea dos verificaciones HTTP cada cinco minutos: `https://<dominio>/health/ready` y `https://<dominio>/`. Una respuesta distinta de 2xx debe generar un aviso al grupo; prueba el canal de avisos antes de la demo.
2. En el VPS, configura una alerta de disco para la particion que contiene los datos de Docker y PostgreSQL. El umbral es 80 % de uso. Comprueba la particion con `df -P /var/lib/docker` y configura el agente de monitoreo del VPS para avisar al mismo grupo.
3. Tras un despliegue, verifica ambas URL desde otra red y confirma que el monitor externo ve el estado nuevo. Revisa tambien `docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml --profile app ps` en el VPS.
4. Ante una caida, comprueba primero `/health/ready`, luego el estado de Compose y los logs de `gateway`, `backend` y `postgres`. Si empezo justo despues de un despliegue, sigue `docs/runbooks/deploy.md` para volver a la revision anterior.

El monitor externo y la alerta de disco aun requieren las cuentas, el dominio y el VPS. Una respuesta local correcta no sustituye esa verificacion.
