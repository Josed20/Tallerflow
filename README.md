# TallerFlow

Cimientos seguros de los Sprints 1 y 2 para la plataforma de trazabilidad de talleres de confección.

El recorrido integrado permite crear desde la web el primer taller y su `OWNER`, entrar directamente con una sesión segura, consultar el usuario y taller actuales, cerrar la sesión y aislar los datos por taller. El bootstrap por CLI del Sprint 1 se conserva para operación y recuperación.

## Requisitos

- Docker Desktop con Docker Compose v2.
- Go 1.27.x.
- Node.js 24 y npm.
- Git.

PostgreSQL y Flyway se ejecutan en contenedores; no necesitan instalación local.

## Inicio desde cero

1. Crea la configuración local y cambia sus valores si el entorno no es descartable:

   ```powershell
   Copy-Item .env.example .env
   ```

2. Inicia PostgreSQL y aplica las migraciones:

   ```powershell
   docker compose up -d --wait postgres flyway
   docker compose run --rm flyway validate
   ```

3. Construye e inicia la aplicación completa:

   ```powershell
   docker compose -f compose.yaml -f compose.mail.yaml --profile app up -d --build --wait
   ```

4. Abre [http://localhost:8080](http://localhost:8080). Si la instalación está vacía, TallerFlow muestra automáticamente el formulario para crear el primer taller y su `OWNER`. La contraseña ingresada es definitiva y la sesión comienza al terminar.

Como alternativa operativa, todavía puedes crear el único `OWNER` inicial por CLI. La contraseña se lee por stdin y no aparece como argumento ni en la salida:

   ```powershell
   $bootstrapPassword = Read-Host 'Contraseña temporal del OWNER' -AsSecureString
   $pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($bootstrapPassword)
   try {
       $plainPassword = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer)
       $plainPassword | docker compose --profile tools run --rm -T bootstrap `
           --password-stdin `
           --email owner@tallerflow.pe `
           --name 'Owner TallerFlow' `
           --workshop 'Taller principal'
   }
   finally {
       [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer)
       Remove-Variable plainPassword, bootstrapPassword -ErrorAction SilentlyContinue
   }
   ```

   La primera ejecución devuelve `OWNER_BOOTSTRAPPED`. Cualquier intento posterior se rechaza con `BOOTSTRAP_ALREADY_EXISTS`. El primer login obliga a cambiar la contraseña.

Caddy sirve Vue y la API bajo el mismo origen. Los health checks son `/health/live` y `/health/ready`.

## Verificación completa del Sprint 2

En Windows:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-sprint2.ps1
```

En Linux o CI:

```sh
./scripts/verify-sprint2.sh
```

La verificación ejecuta primero la regresión integral del Sprint 1 (bootstrap CLI, login, cambio obligatorio, CSRF, logout y throttling). Después parte tres veces de volúmenes vacíos y ejecuta pruebas Go con detector de carreras y análisis estático, unitarias y build de Vue, contratos OpenAPI y SQL, una carrera real de dos altas contra PostgreSQL y el recorrido Chromium de onboarding a 360 px. Confirma que queda un solo grafo `OWNER`, una sola sesión, ningún secreto en el almacenamiento del navegador y que una instalación reclamada ya no permite repetir el alta.

Usa el proyecto Compose aislado `tallerflow_sprint2_verify`, PostgreSQL en `127.0.0.1:55434` y la aplicación en `http://localhost:18081`. Al terminar elimina únicamente sus contenedores y volumen descartable.

## Verificación completa del Sprint 1

En Windows:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-sprint1.ps1
```

En Linux o CI:

```sh
./scripts/verify-sprint1.sh
```

Los scripts ejecutan:

- pruebas Go con detector de carreras en Linux;
- pruebas unitarias y build de Vue;
- Flyway y contratos SQL;
- constraints, grants y aislamiento RLS;
- bootstrap único y login real contra PostgreSQL;
- imágenes de producción y health checks;
- E2E Chromium a 360 px para login, cambio obligatorio, rutas privadas, CSRF, logout y rate limit.

La verificación usa el proyecto Compose aislado `tallerflow_sprint1_verify`, PostgreSQL en `127.0.0.1:55432` y la aplicación en `http://localhost:18080`. Al terminar elimina únicamente sus contenedores y volumen descartable. No reutiliza ni borra el volumen del proyecto Compose normal.

## Pruebas de datos manuales

Con PostgreSQL y Flyway iniciados:

```powershell
Get-Content database/tests/constraints.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow
Get-Content database/tests/identity_contract.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow
Get-Content database/tests/rls_setup.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow
Get-Content database/tests/rls.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_app -d tallerflow
Get-Content database/tests/password_reset_contract.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_bootstrap -d tallerflow
Get-Content database/tests/password_reset_isolation.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U tallerflow_app -d tallerflow
```

## Detener el entorno

Conservar los datos locales:

```powershell
docker compose -f compose.yaml -f compose.mail.yaml --profile app --profile tools down --remove-orphans
```

Eliminar también la base local descartable:

```powershell
docker compose -f compose.yaml -f compose.mail.yaml --profile app --profile tools down --volumes --remove-orphans
```

`down --volumes` destruye la base del proyecto Compose seleccionado. No se debe ejecutar sobre datos que se necesiten conservar.

## Roles de PostgreSQL

| Rol | Uso |
|---|---|
| `tallerflow_owner` | Propiedad de la base; no lo usa la aplicación. |
| `tallerflow_migrator` | Flyway y cambios versionados de esquema. |
| `tallerflow_app` | API con privilegios limitados y RLS. |
| `tallerflow_bootstrap` | Creación transaccional y única del primer OWNER. |

Las operaciones privadas se ejecutan en una transacción que establece el taller con `SET LOCAL app.workshop_id`. Sin ese contexto, las tablas protegidas no devuelven filas.

## Migraciones y producción

- Agrega migraciones en `database/migrations/` con formato `V<n>__descripcion.sql`.
- No edites una migración ya compartida o ejecutada.
- El backend no usa `AutoMigrate`.
- Los valores de `.env.example` son solo de desarrollo.
- En producción se inyectan secretos únicos desde el sistema de despliegue.

Valida la superposición de producción con:

```powershell
docker compose -f compose.yaml -f compose.production.yaml --profile app config
```

## Documentación

- [Contexto del producto](docs/contexto-proyecto.md)
- [Plan del Sprint 1](docs/sprints/sprint-01-team-plan.md)
- [Handoff operativo](docs/sprints/sprint-01-human-handoff.md)
- [Contrato OpenAPI](docs/contracts/openapi.yaml)
