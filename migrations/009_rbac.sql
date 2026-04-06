-- Migration: Create RBAC tables (permissions, roles, role_permissions, user_roles)
-- and seed default permissions and system roles
-- Depends on: 008_email_system (applications and users tables must exist)

-- ─── permissions ──────────────────────────────────────────────────────────────

CREATE TABLE permissions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    resource    VARCHAR(100) NOT NULL,
    action      VARCHAR(100) NOT NULL,
    description TEXT                  DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT idx_permission_resource_action UNIQUE (resource, action)
);

-- ─── roles ────────────────────────────────────────────────────────────────────

CREATE TABLE roles (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id      UUID         NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    description TEXT                  DEFAULT '',
    is_system   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT idx_role_app_name UNIQUE (app_id, name)
);

CREATE INDEX idx_roles_app_id ON roles(app_id);

-- ─── role_permissions ─────────────────────────────────────────────────────────

CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id)       ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- ─── user_roles ───────────────────────────────────────────────────────────────

CREATE TABLE user_roles (
    user_id     UUID        NOT NULL REFERENCES users(id)        ON DELETE CASCADE,
    role_id     UUID        NOT NULL REFERENCES roles(id)        ON DELETE CASCADE,
    app_id      UUID        NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by UUID,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id   ON user_roles(user_id);
CREATE INDEX idx_user_roles_app_id    ON user_roles(app_id);
CREATE INDEX idx_user_role_app_user   ON user_roles(app_id, user_id);

-- ============================================================
-- Seed default permissions (global, not per-app)
-- ============================================================
INSERT INTO permissions (id, resource, action, description) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'user',       'read',   'View user profiles and information'),
    ('a0000000-0000-0000-0000-000000000002', 'user',       'write',  'Create and update user information'),
    ('a0000000-0000-0000-0000-000000000003', 'user',       'delete', 'Delete user accounts'),
    ('a0000000-0000-0000-0000-000000000004', 'log',        'read',   'View activity logs'),
    ('a0000000-0000-0000-0000-000000000005', 'log',        'delete', 'Delete activity log entries'),
    ('a0000000-0000-0000-0000-000000000006', 'settings',   'read',   'View application settings'),
    ('a0000000-0000-0000-0000-000000000007', 'settings',   'write',  'Modify application settings'),
    ('a0000000-0000-0000-0000-000000000008', 'role',       'read',   'View roles and permissions'),
    ('a0000000-0000-0000-0000-000000000009', 'role',       'write',  'Create, update, and delete roles'),
    ('a0000000-0000-0000-0000-000000000010', 'role',       'assign', 'Assign and revoke roles for users')
ON CONFLICT (resource, action) DO NOTHING;

-- ============================================================
-- Create default roles for EACH existing application
-- ============================================================

-- Admin role: all permissions
INSERT INTO roles (id, app_id, name, description, is_system, created_at, updated_at)
SELECT
    gen_random_uuid(),
    a.id,
    'admin',
    'Full access to all resources within the application',
    TRUE,
    NOW(),
    NOW()
FROM applications a
WHERE NOT EXISTS (
    SELECT 1 FROM roles r WHERE r.app_id = a.id AND r.name = 'admin'
);

-- Member role: standard user access
INSERT INTO roles (id, app_id, name, description, is_system, created_at, updated_at)
SELECT
    gen_random_uuid(),
    a.id,
    'member',
    'Standard user with read and limited write access',
    TRUE,
    NOW(),
    NOW()
FROM applications a
WHERE NOT EXISTS (
    SELECT 1 FROM roles r WHERE r.app_id = a.id AND r.name = 'member'
);

-- Viewer role: read-only access
INSERT INTO roles (id, app_id, name, description, is_system, created_at, updated_at)
SELECT
    gen_random_uuid(),
    a.id,
    'viewer',
    'Read-only access to resources',
    TRUE,
    NOW(),
    NOW()
FROM applications a
WHERE NOT EXISTS (
    SELECT 1 FROM roles r WHERE r.app_id = a.id AND r.name = 'viewer'
);

-- ============================================================
-- Assign permissions to roles
-- ============================================================

-- Admin gets ALL permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin' AND r.is_system = TRUE
ON CONFLICT DO NOTHING;

-- Member gets: user:read, user:write, log:read, role:read, settings:read, settings:write
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'member' AND r.is_system = TRUE
  AND (
    (p.resource = 'user' AND p.action IN ('read', 'write'))
    OR (p.resource = 'log' AND p.action = 'read')
    OR (p.resource = 'role' AND p.action = 'read')
    OR (p.resource = 'settings' AND p.action IN ('read', 'write'))
  )
ON CONFLICT DO NOTHING;

-- Viewer gets: user:read, log:read, role:read
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'viewer' AND r.is_system = TRUE
  AND (
    (p.resource = 'user' AND p.action = 'read')
    OR (p.resource = 'log' AND p.action = 'read')
    OR (p.resource = 'role' AND p.action = 'read')
  )
ON CONFLICT DO NOTHING;
