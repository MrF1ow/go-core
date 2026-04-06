-- ─── session_groups ───────────────────────────────────────────────────────────

CREATE TABLE session_groups (
    id            UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID    NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name          TEXT    NOT NULL,
    description   TEXT    NOT NULL DEFAULT '',
    global_logout BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_groups_tenant_id ON session_groups(tenant_id);

-- ─── session_group_apps ───────────────────────────────────────────────────────

CREATE TABLE session_group_apps (
    session_group_id UUID        NOT NULL REFERENCES session_groups(id) ON DELETE CASCADE,
    app_id           UUID        NOT NULL REFERENCES applications(id)   ON DELETE CASCADE,
    added_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_group_id, app_id)
);

CREATE UNIQUE INDEX idx_session_group_app_id ON session_group_apps(app_id);
