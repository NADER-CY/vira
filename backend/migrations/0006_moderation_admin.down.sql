DELETE FROM authz.admin_role_bindings WHERE role_key = 'moderator';
DELETE FROM authz.role_permissions WHERE permission_key IN ('users.read', 'users.write', 'reports.read', 'reports.write')
   OR role_key = 'moderator';
DELETE FROM authz.roles WHERE key = 'moderator';
DELETE FROM authz.permissions WHERE key IN ('users.read', 'users.write', 'reports.read', 'reports.write');
DROP TABLE moderation.reports;
REVOKE USAGE ON SCHEMA moderation FROM app_public, app_admin;
