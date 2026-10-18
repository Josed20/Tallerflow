# Sprint 3: primera entrega de Stephano

Esta entrega prepara la configuracion que consumiran registro, consola y Android. La aplicacion web conserva el transporte actual: rutas relativas, cookie de sesion y CSRF. La API movil se habilitara cuando Jose entregue el contrato bearer y el token se guarde de forma cifrada en Android.

## Configuracion

`.env.example` contiene los valores de desarrollo. En produccion, define `DB_CONSOLE_PASSWORD`, `TF_CONSOLE_DATABASE_URL` y `TF_CONSOLE_TOTP_KEY` con secretos diferentes de los ejemplos; el URL de consola debe usar el mismo password que el rol PostgreSQL. `TF_MOBILE_ORIGINS`, `TF_PUBLIC_BASE_URL`, `TF_SIGNUP_ENABLED` y `TF_CONSOLE_ORIGIN` se pasan al backend para que sus modulos los consuman al integrarse.

`VITE_API_BASE_URL` es una variable de **compilacion** del frontend. Dejala vacia para la web (incluido Docker y Vite local). Al compilar Android, asigna la URL publica HTTPS, por ejemplo `https://tallerflow.pe`. No es un secreto: quedara dentro del APK. La URL del backend para Docker sigue siendo una variable de runtime distinta.

`tallerflow_console` recibe `CONNECT` a la base y `NOBYPASSRLS`; no recibe `SELECT` ni otro permiso sobre las tablas privadas. Las funciones y permisos especificos de la consola pertenecen a la migracion de Michael.

El script `init-roles.sh` se ejecuta automaticamente solo al crear un volumen PostgreSQL vacio. Para agregar el rol a una base local que ya tiene ordenes y usuarios, actualiza `DB_CONSOLE_PASSWORD` en tu `.env`, reinicia solo PostgreSQL para pasar la variable al contenedor y ejecuta el script idempotente:

```powershell
docker compose up -d postgres
docker compose exec -T postgres sh /docker-entrypoint-initdb.d/10-init-roles.sh
```

Esto conserva el volumen y los datos existentes. Si el rol ya existe, el script no cambia su password; coordina una rotacion explicita antes de cambiar la URL de conexion.

## Cliente HTTP

El cliente compartido esta en `frontend/src/modules/session/api.ts`. En la web se usa sin configuracion adicional. Android configurara `configureApiTransport({ kind: 'mobile', getToken, onUnauthorized })` durante el arranque. `getToken` devolvera el token desde el almacenamiento seguro; hasta entonces no se persiste ningun token. En modo movil las peticiones omiten cookies y CSRF, agregan bearer cuando existe un token y notifican un `401` para limpiar la sesion. Los errores del backend conservan su codigo y request ID.

## Comprobacion local

```powershell
docker compose --profile app config --quiet
npm --prefix frontend test
npm --prefix frontend run build
docker compose --profile app up -d --build --wait
docker compose --profile app ps
```

Abre `http://localhost:8080` y entra con tu cuenta actual. La web debe seguir funcionando igual. Para confirmar los permisos del rol:

```powershell
docker compose exec -T postgres psql -U postgres -d tallerflow -c "SELECT rolname, rolbypassrls, has_table_privilege('tallerflow_console', 'public.orders', 'SELECT') AS reads_orders FROM pg_roles WHERE rolname = 'tallerflow_console';"
```

El resultado esperado es `rolbypassrls = f` y `reads_orders = f`. La prueba automatizada del cliente HTTP simula el futuro modo Android, sin requerir que la API bearer de Jose ya exista.
