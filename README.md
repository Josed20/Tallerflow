# TallerFlow

Base técnica del Sprint 1 para la plataforma de trazabilidad de talleres de confección.

## Requisitos

- Docker Desktop con Docker Compose v2.
- Git.

No se requiere instalar PostgreSQL ni Flyway localmente.

## Base de datos desde cero

1. Crea la configuración local:

   ```powershell
   Copy-Item .env.example .env
   ```

2. Cambia las contraseñas de `.env` si el entorno no es exclusivamente local.
   Si el puerto `5432` ya está ocupado, cambia `POSTGRES_PORT` en ese archivo.

3. Inicia PostgreSQL y ejecuta las migraciones:

   ```powershell
   docker compose up --wait postgres flyway
   ```

4. Valida el historial de Flyway:

   ```powershell
   docker compose run --rm flyway validate
   docker compose run --rm flyway info
   ```

Para repetir la prueba con un volumen vacío:

```powershell
docker compose down --volumes
docker compose up --wait postgres flyway
```

`down --volumes` elimina la base local de Docker. No debe usarse sobre datos que se necesiten conservar.

## Pruebas de datos

Con PostgreSQL y Flyway iniciados:

```powershell
Get-Content database/tests/constraints.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d tallerflow
Get-Content database/tests/rls.sql -Raw | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d tallerflow
```

Las pruebas verifican constraints básicos y que RLS no muestre talleres sin contexto ni permita ver otro taller.

## Aplicación completa

Cuando existan `backend/go.mod` y `frontend/package-lock.json`, inicia todos los servicios:

```powershell
docker compose --profile app up --build --wait
```

La aplicación quedará disponible en `http://localhost:8080`. Caddy sirve frontend y API bajo el mismo origen.

## Roles de PostgreSQL

| Rol | Uso |
|---|---|
| `tallerflow_owner` | Propiedad de la base; no lo usa la aplicación. |
| `tallerflow_migrator` | Flyway y cambios versionados de esquema. |
| `tallerflow_app` | Conexiones del backend con privilegios limitados y RLS. |

La aplicación debe abrir cada operación privada en una transacción y establecer el taller con:

```sql
SET LOCAL app.workshop_id = '<uuid-del-taller>';
```

Sin ese contexto, las tablas protegidas por tenant no devuelven filas.

## Migraciones

- Agrega archivos nuevos en `database/migrations/` con formato `V<n>__descripcion.sql`.
- No edites una migración que ya se haya fusionado o ejecutado en un entorno compartido.
- Stephano coordina la numeración de Flyway durante el Sprint 1.
- El backend no debe usar AutoMigrate.

## Producción

`compose.production.yaml` complementa el Compose base y evita publicar PostgreSQL en el host:

```powershell
docker compose -f compose.yaml -f compose.production.yaml --profile app config
```

Los valores de `.env.example` son solo para desarrollo. En producción se deben inyectar secretos únicos desde el sistema de despliegue.

## Documentación

- [Contexto del producto](docs/contexto-proyecto.md)
- [Plan del Sprint 1](sprint-01-team-plan.md)
