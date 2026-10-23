# Monitoreo de TallerFlow

La puesta en marcha requiere el dominio HTTPS y acceso al VPS de Jose.

1. En el servicio de monitoreo externo, crea dos verificaciones HTTP cada cinco minutos: `https://<dominio>/health/ready` y `https://<dominio>/`. Una respuesta distinta de 2xx debe generar un aviso al grupo; prueba el canal de avisos antes de la demo.
2. En el VPS, instala la alerta de disco (ver la seccion siguiente). Avisa por correo cuando la particion de `/var/lib/docker` pasa el 80 % de uso.
3. Tras un despliegue, verifica ambas URL desde otra red y confirma que el monitor externo ve el estado nuevo. Revisa tambien `docker compose -f compose.yaml -f compose.production.yaml -f compose.tunnel.yaml --profile app ps` en el VPS.
4. Ante una caida, comprueba primero `/health/ready`, luego el estado de Compose y los logs de `gateway`, `backend` y `postgres`. Si empezo justo despues de un despliegue, sigue `docs/runbooks/deploy.md` para volver a la revision anterior.

El monitor externo y la alerta de disco aun requieren las cuentas, el dominio y el VPS. Una respuesta local correcta no sustituye esa verificacion.

## Alerta de disco

`scripts/disk-alert.sh` revisa cada hora el uso de la particion de `/var/lib/docker`. Si pasa el umbral, manda un correo con la misma configuracion SMTP de la API (`TF_SMTP_*` del `.env`), asi que no hace falta otro servicio de correo. Mientras el disco siga lleno, repite el aviso como maximo cada 12 horas.

1. Crea `/etc/tallerflow/disk-alert.env` (dueño `root`, modo `0600`). El destinatario no va en el repositorio porque es publico:

   ```dotenv
   DISK_ALERT_TO=<correo que recibe los avisos>
   DISK_ALERT_THRESHOLD=80
   ```

2. Copia `infra/systemd/tallerflow-disk-alert.service` y `.timer` a `/etc/systemd/system/` y ejecuta `sudo systemctl daemon-reload`.
3. Manda un correo de prueba, aunque el disco este bajo el umbral:

   ```bash
   sudo systemd-run --wait --pipe --uid=deploy \
     --property=EnvironmentFile=/opt/tallerflow/.env \
     --property=EnvironmentFile=/etc/tallerflow/disk-alert.env \
     /usr/bin/bash /opt/tallerflow/scripts/disk-alert.sh --test
   ```

   Confirma que llego a la bandeja de entrada y no a spam.
4. Activa el timer con `sudo systemctl enable --now tallerflow-disk-alert.timer` y revisa `systemctl list-timers tallerflow-disk-alert.timer`. El resultado de cada revision queda en `journalctl -u tallerflow-disk-alert.service`.

Para cambiar el destinatario, edita `DISK_ALERT_TO`; el siguiente aviso ya usa el nuevo valor.
