-- Migration: Operator IAM evidence tables and GUI account disable.
-- Depends on: 016_operator_rbac, 017_admin_account_operator_role

ALTER TABLE admin_accounts
    ADD COLUMN disabled_at TIMESTAMPTZ;

CREATE TABLE operator_iam_events (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    actor_kind         VARCHAR(32)  NOT NULL,
    actor_key_id       UUID         REFERENCES api_keys(id) ON DELETE SET NULL,
    actor_account_id   UUID         REFERENCES admin_accounts(id) ON DELETE SET NULL,
    target_kind        VARCHAR(32)  NOT NULL,
    target_key_id      UUID         REFERENCES api_keys(id) ON DELETE SET NULL,
    target_account_id  UUID         REFERENCES admin_accounts(id) ON DELETE SET NULL,
    old_role_id        UUID         REFERENCES operator_roles(id) ON DELETE RESTRICT,
    new_role_id        UUID         REFERENCES operator_roles(id) ON DELETE RESTRICT,
    action             VARCHAR(32)  NOT NULL,
    CONSTRAINT operator_iam_events_actor_kind_check CHECK (actor_kind IN ('env_key', 'api_key', 'gui_account', 'setup_cli')),
    CONSTRAINT operator_iam_events_target_kind_check CHECK (target_kind IN ('env_key', 'api_key', 'gui_account')),
    CONSTRAINT operator_iam_events_action_check CHECK (action IN ('assign', 'create_principal', 'revoke_key', 'disable_principal'))
);

CREATE INDEX idx_operator_iam_events_at ON operator_iam_events (at DESC);
CREATE INDEX idx_operator_iam_events_target_key_id ON operator_iam_events (target_key_id);
CREATE INDEX idx_operator_iam_events_target_account_id ON operator_iam_events (target_account_id);

CREATE TABLE operator_access_logs (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    kind        VARCHAR(32)  NOT NULL,
    key_id      UUID         REFERENCES api_keys(id) ON DELETE SET NULL,
    account_id  UUID         REFERENCES admin_accounts(id) ON DELETE SET NULL,
    role_name   VARCHAR(100) NOT NULL DEFAULT '',
    method      VARCHAR(16)  NOT NULL,
    path        TEXT         NOT NULL,
    decision    VARCHAR(16)  NOT NULL,
    resource    VARCHAR(100) NOT NULL,
    action      VARCHAR(100) NOT NULL,
    status      INTEGER      NOT NULL,
    CONSTRAINT operator_access_logs_kind_check CHECK (kind IN ('env_key', 'api_key', 'gui_account')),
    CONSTRAINT operator_access_logs_decision_check CHECK (decision IN ('allow', 'deny'))
);

CREATE INDEX idx_operator_access_logs_at ON operator_access_logs (at DESC);
CREATE INDEX idx_operator_access_logs_decision ON operator_access_logs (decision);
