-- 0006 moderation reports (P2) and Phase 2 admin permissions (user management, report review).

CREATE TABLE moderation.reports (
  id              uuid        PRIMARY KEY,
  reporter_id     uuid        NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
  target_type     text        NOT NULL CHECK (target_type IN ('post', 'comment', 'user')),
  target_id       uuid        NOT NULL,
  target_owner_id uuid        REFERENCES identity.users (id) ON DELETE SET NULL,
  reason          text        NOT NULL CHECK (reason IN ('spam', 'harassment', 'hate', 'violence', 'nudity', 'scam',
                                                          'counterfeit', 'misinformation', 'self_harm', 'other')),
  details         text        NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  state           text        NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'actioned', 'dismissed')),
  resolution      text        CHECK (resolution IN ('dismiss', 'remove_content', 'restrict_user', 'suspend_user')),
  resolution_note text        CHECK (char_length(resolution_note) <= 500),
  resolved_by     uuid,       -- authz.admin_users (separate realm; no FK by design, ADR-016)
  created_at      timestamptz NOT NULL DEFAULT now(),
  resolved_at     timestamptz
);
-- One open report per reporter and target (duplicate submissions return the existing report).
CREATE UNIQUE INDEX reports_one_open ON moderation.reports (reporter_id, target_type, target_id) WHERE state = 'open';
CREATE INDEX reports_queue ON moderation.reports (state, created_at DESC);
CREATE INDEX reports_target ON moderation.reports (target_type, target_id);
CREATE INDEX reports_owner ON moderation.reports (target_owner_id, created_at DESC);

GRANT USAGE ON SCHEMA moderation TO app_public, app_admin;
GRANT SELECT, INSERT ON moderation.reports TO app_public;
GRANT SELECT, UPDATE ON moderation.reports TO app_admin;

INSERT INTO authz.permissions (key, description) VALUES
  ('users.read',   'View users, profiles, sessions and status history'),
  ('users.write',  'Change user account status and revoke sessions'),
  ('reports.read', 'View moderation reports'),
  ('reports.write','Resolve moderation reports and remove content');

INSERT INTO authz.roles (key, name_fa, description) VALUES
  ('moderator', 'ناظر محتوا', 'Review reports, remove content, restrict users');

INSERT INTO authz.role_permissions (role_key, permission_key) VALUES
  ('super_admin', 'users.read'), ('super_admin', 'users.write'), ('super_admin', 'reports.read'), ('super_admin', 'reports.write'),
  ('moderator', 'system.read'), ('moderator', 'users.read'), ('moderator', 'users.write'),
  ('moderator', 'reports.read'), ('moderator', 'reports.write'),
  ('auditor', 'users.read'), ('auditor', 'reports.read'),
  ('operator', 'users.read');
