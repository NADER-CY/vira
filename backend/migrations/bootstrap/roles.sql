-- VIRA cluster bootstrap: database roles (doc 06 §1). Run ONCE per PostgreSQL cluster by a superuser,
-- before the first migration. Idempotent. Passwords are NOT set here; the environment sets them
-- (deploy/postgres/init/10-roles.sh for local development, the secret store for staging/production).
--
--   migrator           owns every object; used only by the migrate runtime, never at request time
--   app_public         public API (mobile + seller web); subject to row-level security
--   app_admin          Super Admin API; BYPASSRLS, every action audited at the application layer
--   app_worker         background worker; no BYPASSRLS, uses audience 'system' policies
--   app_ledger_writer  NOLOGIN; assumed with SET LOCAL ROLE inside ledger transaction scripts (Phase 6)
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'migrator') THEN
    CREATE ROLE migrator LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_public') THEN
    CREATE ROLE app_public LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_admin') THEN
    CREATE ROLE app_admin LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE BYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_worker') THEN
    CREATE ROLE app_worker LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_ledger_writer') THEN
    CREATE ROLE app_ledger_writer NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
END
$$;
