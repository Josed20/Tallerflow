#!/bin/sh
set -eu

required_vars="DB_OWNER_USER DB_OWNER_PASSWORD DB_MIGRATION_USER DB_MIGRATION_PASSWORD DB_APP_USER DB_APP_PASSWORD POSTGRES_DB"
for variable_name in $required_vars; do
  eval "variable_value=\${$variable_name:-}"
  if [ -z "$variable_value" ]; then
    echo "Missing required environment variable: $variable_name" >&2
    exit 1
  fi
done

psql --username "$POSTGRES_USER" --dbname postgres \
  --set=owner_user="$DB_OWNER_USER" \
  --set=owner_password="$DB_OWNER_PASSWORD" \
  --set=migration_user="$DB_MIGRATION_USER" \
  --set=migration_password="$DB_MIGRATION_PASSWORD" \
  --set=app_user="$DB_APP_USER" \
  --set=app_password="$DB_APP_PASSWORD" \
  --set=app_database="$POSTGRES_DB" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L NOINHERIT', :'owner_user', :'owner_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'owner_user') \gexec
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L NOINHERIT', :'migration_user', :'migration_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'migration_user') \gexec
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L NOINHERIT', :'app_user', :'app_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'app_user') \gexec
SELECT format('ALTER DATABASE %I OWNER TO %I', :'app_database', :'owner_user') \gexec
SELECT format('REVOKE ALL ON DATABASE %I FROM PUBLIC', :'app_database') \gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO %I, %I', :'app_database', :'migration_user', :'app_user') \gexec
SQL

psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  --set=owner_user="$DB_OWNER_USER" \
  --set=migration_user="$DB_MIGRATION_USER" <<'SQL'
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
SELECT format('ALTER SCHEMA public OWNER TO %I', :'migration_user') \gexec
SELECT format('GRANT USAGE, CREATE ON SCHEMA public TO %I', :'migration_user') \gexec
SQL

