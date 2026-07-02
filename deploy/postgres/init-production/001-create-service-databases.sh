#!/bin/sh
set -eu

required_vars="
AUTH_DB_PASSWORD
FILE_DB_PASSWORD
KNOWLEDGE_DB_PASSWORD
QA_DB_PASSWORD
DOCUMENT_DB_PASSWORD
AI_GATEWAY_DB_PASSWORD
"

for name in $required_vars; do
  eval "value=\${$name:-}"
  if [ -z "$value" ]; then
    echo "$name is required for production database initialization" >&2
    exit 1
  fi
done

create_role_and_database() {
  role="$1"
  password="$2"
  database="$3"

  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
    --set=role_name="$role" \
    --set=role_password="$password" \
    --set=database_name="$database" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'role_name', :'role_password')
WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = :'role_name')
\gexec
SELECT format('ALTER ROLE %I WITH LOGIN PASSWORD %L', :'role_name', :'role_password')
WHERE EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = :'role_name')
\gexec
SELECT format('CREATE DATABASE %I OWNER %I', :'database_name', :'role_name')
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'database_name')
\gexec
SQL
}

create_role_and_database auth_app "$AUTH_DB_PASSWORD" auth_system
create_role_and_database file_app "$FILE_DB_PASSWORD" file_system
create_role_and_database knowledge_app "$KNOWLEDGE_DB_PASSWORD" knowledge_system
create_role_and_database qa_app "$QA_DB_PASSWORD" qa_system
create_role_and_database document_app "$DOCUMENT_DB_PASSWORD" document_system
create_role_and_database ai_gateway_app "$AI_GATEWAY_DB_PASSWORD" ai_gateway_system
