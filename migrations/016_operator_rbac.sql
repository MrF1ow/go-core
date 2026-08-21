-- Migration: Operator IAM catalog, role column on api_keys, backfill existing admin keys.
-- Depends on: 006_api_keys

-- ─── operator_permissions ─────────────────────────────────────────────────────

CREATE TABLE operator_permissions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    resource    VARCHAR(100) NOT NULL,
    action      VARCHAR(100) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT idx_operator_permission_resource_action UNIQUE (resource, action)
);

-- ─── operator_roles ───────────────────────────────────────────────────────────

CREATE TABLE operator_roles (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    is_system   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT idx_operator_roles_name UNIQUE (name)
);

-- ─── operator_role_permissions ────────────────────────────────────────────────

CREATE TABLE operator_role_permissions (
    role_id       UUID NOT NULL REFERENCES operator_roles(id)       ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES operator_permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Seed catalog. IDs are stable so grant SQL can name them.
-- Keep in sync with internal/operator/catalog.go.

INSERT INTO operator_permissions (id, resource, action, description) VALUES
    ('c0000000-0000-0000-0000-000000000001', 'dashboard',      'read',  'View admin GUI home and stats'),
    ('c0000000-0000-0000-0000-000000000002', 'tenants',        'read',  'View tenants'),
    ('c0000000-0000-0000-0000-000000000003', 'tenants',        'write', 'Create, update, and delete tenants'),
    ('c0000000-0000-0000-0000-000000000004', 'applications',   'read',  'View applications'),
    ('c0000000-0000-0000-0000-000000000005', 'applications',   'write', 'Create, update, and delete applications'),
    ('c0000000-0000-0000-0000-000000000006', 'oauth',          'read',  'View OAuth provider configs'),
    ('c0000000-0000-0000-0000-000000000007', 'oauth',          'write', 'Change OAuth provider configs'),
    ('c0000000-0000-0000-0000-000000000008', 'oidc',           'read',  'View OIDC clients'),
    ('c0000000-0000-0000-0000-000000000009', 'oidc',           'write', 'Change OIDC clients'),
    ('c0000000-0000-0000-0000-000000000010', 'session_groups', 'read',  'View session groups'),
    ('c0000000-0000-0000-0000-000000000011', 'session_groups', 'write', 'Change session groups'),
    ('c0000000-0000-0000-0000-000000000012', 'users',          'read',  'View app users'),
    ('c0000000-0000-0000-0000-000000000013', 'users',          'write', 'Change app users, import, devices'),
    ('c0000000-0000-0000-0000-000000000014', 'sessions',       'read',  'View app-user sessions'),
    ('c0000000-0000-0000-0000-000000000015', 'sessions',       'write', 'Revoke app-user sessions'),
    ('c0000000-0000-0000-0000-000000000016', 'ip_rules',       'read',  'View IP rules'),
    ('c0000000-0000-0000-0000-000000000017', 'ip_rules',       'write', 'Change IP rules'),
    ('c0000000-0000-0000-0000-000000000018', 'end_user_rbac',  'read',  'View app-user roles'),
    ('c0000000-0000-0000-0000-000000000019', 'end_user_rbac',  'write', 'Change app-user roles'),
    ('c0000000-0000-0000-0000-000000000020', 'email',          'read',  'View email servers and templates'),
    ('c0000000-0000-0000-0000-000000000021', 'email',          'write', 'Change email config and send'),
    ('c0000000-0000-0000-0000-000000000022', 'logs',           'read',  'View end-user activity logs'),
    ('c0000000-0000-0000-0000-000000000023', 'api_keys',       'read',  'View API keys'),
    ('c0000000-0000-0000-0000-000000000024', 'api_keys',       'write', 'Create and revoke API keys'),
    ('c0000000-0000-0000-0000-000000000025', 'webhooks',       'read',  'View webhooks'),
    ('c0000000-0000-0000-0000-000000000026', 'webhooks',       'write', 'Change webhooks'),
    ('c0000000-0000-0000-0000-000000000027', 'monitoring',     'read',  'View health and metrics'),
    ('c0000000-0000-0000-0000-000000000028', 'settings',       'read',  'View system settings'),
    ('c0000000-0000-0000-0000-000000000029', 'settings',       'write', 'Change system settings'),
    ('c0000000-0000-0000-0000-000000000030', 'admin_iam',      'read',  'View operator roles and assignments'),
    ('c0000000-0000-0000-0000-000000000031', 'admin_iam',      'write', 'Change operator roles and assignments')
ON CONFLICT (resource, action) DO NOTHING;

INSERT INTO operator_roles (id, name, description, is_system) VALUES
    ('d0000000-0000-0000-0000-000000000001', 'superadmin', 'Full operator access including IAM', TRUE),
    ('d0000000-0000-0000-0000-000000000002', 'admin',      'Full product access. No operator IAM.', TRUE),
    ('d0000000-0000-0000-0000-000000000003', 'support',    'Users, sessions, logs, and dashboard', TRUE),
    ('d0000000-0000-0000-0000-000000000004', 'viewer',     'Read-only dashboard, users, logs, monitoring', TRUE)
ON CONFLICT (name) DO NOTHING;

INSERT INTO operator_role_permissions (role_id, permission_id)
SELECT 'd0000000-0000-0000-0000-000000000001', id FROM operator_permissions
ON CONFLICT DO NOTHING;

INSERT INTO operator_role_permissions (role_id, permission_id)
SELECT 'd0000000-0000-0000-0000-000000000002', id FROM operator_permissions
WHERE resource <> 'admin_iam'
ON CONFLICT DO NOTHING;

INSERT INTO operator_role_permissions (role_id, permission_id)
SELECT 'd0000000-0000-0000-0000-000000000003', id FROM operator_permissions
WHERE (resource, action) IN (
    ('dashboard', 'read'),
    ('users', 'read'),
    ('users', 'write'),
    ('sessions', 'read'),
    ('sessions', 'write'),
    ('logs', 'read')
)
ON CONFLICT DO NOTHING;

INSERT INTO operator_role_permissions (role_id, permission_id)
SELECT 'd0000000-0000-0000-0000-000000000004', id FROM operator_permissions
WHERE (resource, action) IN (
    ('dashboard', 'read'),
    ('users', 'read'),
    ('logs', 'read'),
    ('monitoring', 'read')
)
ON CONFLICT DO NOTHING;

ALTER TABLE api_keys
    ADD COLUMN operator_role_id UUID REFERENCES operator_roles(id) ON DELETE RESTRICT;

CREATE INDEX idx_api_keys_operator_role_id ON api_keys(operator_role_id);

UPDATE api_keys
SET operator_role_id = 'd0000000-0000-0000-0000-000000000001'
WHERE key_type = 'admin' AND operator_role_id IS NULL;
