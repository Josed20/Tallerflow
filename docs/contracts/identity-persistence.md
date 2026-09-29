# Contrato de persistencia de identidad — Sprint 1

Este contrato une los adaptadores de autenticación, las migraciones Flyway y
la resolución de taller. `database/migrations` es la fuente ejecutable del
esquema; este documento expresa el contrato que no debe romperse entre ramas.

## Conexiones y límites

- La API se conecta como `tallerflow_app`; no es dueña de tablas ni puede
  eludir RLS.
- Flyway se conecta como `tallerflow_migrator` y es el único rol con DDL.
- El bootstrap usa `tallerflow_bootstrap`: conexión separada, sin DDL ni
  `BYPASSRLS`, válida solo para provisionar al OWNER inicial.
- No se almacenan tokens de sesión o CSRF en claro, ni se registran hashes,
  contraseñas o cadenas de conexión.

## Tablas y columnas mínimas

| Tabla | Columnas requeridas |
|---|---|
| `users` | `id uuid PK`, `email citext UNIQUE`, `name text`, `status text`, `created_at timestamptz` |
| `user_credentials` | `user_id uuid PK/FK`, `password_hash text`, `must_change_password boolean`, `password_changed_at timestamptz NULL` |
| `user_sessions` | `id uuid PK`, `user_id uuid FK`, `token_hash bytea UNIQUE`, `csrf_token_hash bytea`, `created_at`, `expires_at`, `revoked_at`, `ip_prefix`, `user_agent` |
| `workshops` | `id uuid PK`, `name text`, `timezone text` |
| `workshop_members` | `id uuid PK`, `user_id uuid FK`, `workshop_id uuid FK`, `role text`, `status text` |
| `audit_events` | `id uuid PK`, `workshop_id uuid FK`, `actor_user_id uuid NULL`, `event_type text`, `details jsonb`, `created_at timestamptz` |

La transacción de bootstrap debe reclamar `bootstrap_state` y después crear
usuario, credencial temporal, taller, membresía `OWNER` y el evento
`OWNER_BOOTSTRAPPED`. Solo una reclamación es válida.
