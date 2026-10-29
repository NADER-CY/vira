#!/bin/sh
# Runs once when the local PostgreSQL volume is first created (after 00-roles.sql created the roles).
# Sets role passwords from the environment and creates the application database owned by migrator.
set -eu
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<SQL
ALTER ROLE migrator  PASSWORD '${MIGRATOR_PASSWORD}';
ALTER ROLE app_public PASSWORD '${APP_PUBLIC_PASSWORD}';
ALTER ROLE app_admin  PASSWORD '${APP_ADMIN_PASSWORD}';
ALTER ROLE app_worker PASSWORD '${APP_WORKER_PASSWORD}';
CREATE DATABASE ${VIRA_DB_NAME} OWNER migrator;
REVOKE ALL ON DATABASE ${VIRA_DB_NAME} FROM PUBLIC;
GRANT CONNECT ON DATABASE ${VIRA_DB_NAME} TO migrator, app_public, app_admin, app_worker;
SQL
