-- Migration: Create webhook_endpoints and webhook_deliveries tables
-- Depends on: 009_rbac (applications table must exist)

-- ─── webhook_endpoints ────────────────────────────────────────────────────────

CREATE TABLE webhook_endpoints (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id     UUID        NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    url        TEXT        NOT NULL,
    secret     TEXT        NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_webhook_app_event         ON webhook_endpoints(app_id, event_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_webhook_endpoints_app_id         ON webhook_endpoints(app_id);
CREATE INDEX idx_webhook_endpoints_deleted_at     ON webhook_endpoints(deleted_at);

-- ─── webhook_deliveries ───────────────────────────────────────────────────────

CREATE TABLE webhook_deliveries (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id   UUID        NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    app_id        UUID        NOT NULL,
    event_type    VARCHAR(64) NOT NULL,
    payload       TEXT        NOT NULL,
    attempt       INT         NOT NULL DEFAULT 1,
    status_code   INT,
    response_body TEXT,
    latency_ms    BIGINT,
    success       BOOLEAN     NOT NULL DEFAULT FALSE,
    error_message TEXT,
    next_retry_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_endpoint_id ON webhook_deliveries(endpoint_id);
CREATE INDEX idx_webhook_deliveries_app_id      ON webhook_deliveries(app_id);
CREATE INDEX idx_webhook_deliveries_event_type  ON webhook_deliveries(event_type);
CREATE INDEX idx_webhook_deliveries_success     ON webhook_deliveries(success);
CREATE INDEX idx_webhook_deliveries_next_retry_at ON webhook_deliveries(next_retry_at) WHERE next_retry_at IS NOT NULL;
