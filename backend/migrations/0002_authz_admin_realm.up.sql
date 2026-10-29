-- 0002 admin realm (ADR-016, ADR-017): admin users are NOT platform users. TOTP is mandatory,
-- sessions last at most 8 h with a 15-minute idle timeout, and every action is audited.

CREATE SCHEMA authz;
COMMENT ON SCHEMA authz IS 'Admin realm accounts, sessions and role bindings; seller-staff bindings arrive in Phase 4';

CREATE TABLE authz.admin_users (
  id               uuid        PRIMARY KEY,
  email            text        NOT NULL CHECK (email = lower(email) AND email ~ '^[^@\s]+@[^@\s]+$'),
  display_name     text        NOT NULL CHECK (length(display_name) BETWEEN 2 AND 80),
  password_hash    text        NOT NULL,
  totp_secret_enc  bytea       NOT NULL,           -- AES-256-GCM, key outside the database
  totp_last_step   bigint      NOT NULL DEFAULT 0, -- replay protection: a TOTP step is accepted once
  status           text        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
  failed_logins    integer     NOT NULL DEFAULT 0,
  locked_until     timestamptz,
  last_login_at    timestamptz,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX admin_users_email_idx ON authz.admin_users (email);

-- Sessions: opaque random token, only its SHA-256 is stored. stage 'mfa' = password checked, TOTP pending.
CREATE TABLE authz.admin_sessions (
  id            uuid        PRIMARY KEY,
  admin_id      uuid        NOT NULL REFERENCES authz.admin_users (id),
  token_hash    bytea       NOT NULL UNIQUE,
  stage         text        NOT NULL CHECK (stage IN ('mfa', 'active')),
  created_at    timestamptz NOT NULL DEFAULT now(),
  last_seen_at  timestamptz NOT NULL DEFAULT now(),
  expires_at    timestamptz NOT NULL,
  revoked_at    timestamptz,
  ip            inet,
  user_agent    text
);
CREATE INDEX admin_sessions_admin_idx ON authz.admin_sessions (admin_id) WHERE revoked_at IS NULL;

CREATE TABLE authz.permissions (
  key         text PRIMARY KEY CHECK (key ~ '^[a-z][a-z_]*\.[a-z_]+$'),
  description text NOT NULL
);

CREATE TABLE authz.roles (
  key         text PRIMARY KEY CHECK (key ~ '^[a-z][a-z_]*$'),
  name_fa     text NOT NULL,
  description text NOT NULL
);

CREATE TABLE authz.role_permissions (
  role_key       text NOT NULL REFERENCES authz.roles (key),
  permission_key text NOT NULL REFERENCES authz.permissions (key),
  PRIMARY KEY (role_key, permission_key)
);

CREATE TABLE authz.admin_role_bindings (
  admin_id   uuid        NOT NULL REFERENCES authz.admin_users (id),
  role_key   text        NOT NULL REFERENCES authz.roles (key),
  granted_at timestamptz NOT NULL DEFAULT now(),
  granted_by text        NOT NULL,
  PRIMARY KEY (admin_id, role_key)
);

INSERT INTO authz.permissions (key, description) VALUES
  ('system.read',  'View service health and platform status'),
  ('flags.read',   'View feature flags'),
  ('flags.write',  'Change non-regulated feature flags'),
  ('config.read',  'View remote configuration'),
  ('audit.read',   'Read the audit log'),
  ('admins.read',  'View admin accounts and roles');

INSERT INTO authz.roles (key, name_fa, description) VALUES
  ('super_admin', 'مدیر ارشد',   'All Phase 1 permissions'),
  ('auditor',     'حسابرس',      'Read-only access including the audit log'),
  ('operator',    'اپراتور',     'Read-only operational access');

INSERT INTO authz.role_permissions (role_key, permission_key)
  SELECT 'super_admin', key FROM authz.permissions;
INSERT INTO authz.role_permissions (role_key, permission_key) VALUES
  ('auditor', 'system.read'), ('auditor', 'flags.read'), ('auditor', 'config.read'),
  ('auditor', 'audit.read'), ('auditor', 'admins.read'),
  ('operator', 'system.read'), ('operator', 'flags.read'), ('operator', 'config.read');

GRANT USAGE ON SCHEMA authz TO app_admin;
GRANT SELECT, INSERT, UPDATE ON authz.admin_users, authz.admin_sessions TO app_admin;
GRANT SELECT ON authz.permissions, authz.roles, authz.role_permissions TO app_admin;
GRANT SELECT, INSERT ON authz.admin_role_bindings TO app_admin;
