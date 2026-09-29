#!/bin/sh
set -eu

required_vars="DB_OWNER_PASSWORD DB_MIGRATION_PASSWORD DB_APP_PASSWORD DB_BOOTSTRAP_PASSWORD POSTGRES_DB"
for variable_name in $required_vars; do
  eval "variable_value=\${$variable_name:-}"
  if [ -z "$variable_value" ]; then
    echo "Missing required environment variable: $variable_name" >&2
    exit 1
  fi
done

psql --username "$POSTGRES_USER" --dbname postgres \
  --set=owner_password="$DB_OWNER_PASSWORD" \
  --set=migration_password="$DB_MIGRATION_PASSWORD" \
  --set=app_password="$DB_APP_PASSWORD" \
  --set=bootstrap_password="$DB_BOOTSTRAP_PASSWORD" \
  --set=app_database="$POSTGRES_DB" <<'SQL'
SELECT format('CREATE ROLE tallerflow_owner LOGIN PASSWORD %L NOINHERIT', :'owner_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'tallerflow_owner') \gexec
SELECT format('CREATE ROLE tallerflow_migrator LOGIN PASSWORD %L NOINHERIT', :'migration_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'tallerflow_migrator') \gexec
SELECT format('CREATE ROLE tallerflow_app LOGIN PASSWORD %L NOINHERIT', :'app_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'tallerflow_app') \gexec
SELECT format('CREATE ROLE tallerflow_bootstrap LOGIN PASSWORD %L NOINHERIT NOBYPASSRLS', :'bootstrap_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'tallerflow_bootstrap') \gexec
SELECT format('ALTER DATABASE %I OWNER TO tallerflow_owner', :'app_database') \gexec
SELECT format('REVOKE ALL ON DATABASE %I FROM PUBLIC', :'app_database') \gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO tallerflow_migrator, tallerflow_app, tallerflow_bootstrap', :'app_database') \gexec
SQL

psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<'SQL'
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
ALTER SCHEMA public OWNER TO tallerflow_migrator;
GRANT USAGE, CREATE ON SCHEMA public TO tallerflow_migrator;
SQL
