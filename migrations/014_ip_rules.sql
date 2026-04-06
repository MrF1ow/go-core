-- ─── ip_rules ─────────────────────────────────────────────────────────────────

CREATE TABLE ip_rules (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id      UUID         NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    rule_type   VARCHAR(10)  NOT NULL,
    match_type  VARCHAR(10)  NOT NULL,
    value       TEXT         NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ip_rule_app ON ip_rules(app_id);
