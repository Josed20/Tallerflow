# Fotos privadas en Cloudflare R2

`V17` conserva las fotos antiguas en `photo_data` y agrega `object_key` para las nuevas. Nunca migres ni borres los registros antiguos durante este sprint. La API sigue entregando `photo_data`: un data URL para fotos antiguas y una URL GET firmada de R2 que vence en 15 minutos para las nuevas. El bucket permanece privado.

## Preparación

1. En la cuenta Cloudflare del equipo, crea un bucket privado **distinto** del bucket de respaldos. Crea credenciales R2 con acceso de lectura, escritura y eliminación solo a este bucket; la eliminación permite limpiar una subida cuyo registro SQL falle. No configures dominio público para el bucket.
2. En el entorno local usa un bucket de prueba, no el de producción. Define `TF_R2_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com`, `TF_R2_BUCKET_PHOTOS`, `TF_R2_ACCESS_KEY_ID` y `TF_R2_SECRET_ACCESS_KEY` en un `.env` privado. Las cuatro variables se requieren juntas; producción no arranca sin ellas.
3. Antes de desplegar, José agrega esas cuatro variables al `/opt/tallerflow/.env` del VPS. Confirma un respaldo del mismo día. La migración `V17` se fusiona **después** de las migraciones `V14`, `V15` y `V16` de Michael y se aplica una sola vez con Flyway.
4. Sube una foto JPEG pequeña en el entorno de prueba, abre la orden y el link de seguimiento, confirma que la imagen aparece, y verifica que el enlace deja de funcionar después de 15 minutos. La respuesta de detalle y seguimiento usa `Cache-Control: no-store`.
5. Comprueba que una foto anterior sigue visible, que otra cuenta/taller no puede consultar la orden, y que la pestaña de consola del navegador no informa bloqueos CSP para la URL de R2.

Para comprobar V17 sin modificar la base de la aplicación, inicia solo PostgreSQL local con `docker compose up -d --wait postgres` y ejecuta `bash scripts/test-r2-migration-local.sh`. El script crea una base temporal, aplica las migraciones anteriores y la elimina al terminar.

Las URLs firmadas actúan como enlaces de acceso temporales. No se registran en logs ni se comparten fuera de la persona que puede ver la orden o el link de seguimiento. Si falla R2, la subida falla antes de escribir la fila; si falla SQL después de subir, se intenta eliminar el objeto sin referencia.
