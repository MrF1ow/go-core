-- Migration: 004_activity_logs
-- Description: Create activity_logs table

CREATE TABLE activity_logs (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id     UUID        NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    timestamp  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address TEXT        NOT NULL DEFAULT '',
    user_agent TEXT        NOT NULL DEFAULT '',
    details    JSONB,
    severity   VARCHAR(20) NOT NULL DEFAULT 'INFORMATIONAL',
    expires_at TIMESTAMPTZ,
    is_anomaly BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_activity_logs_app_id        ON activity_logs(app_id);
CREATE INDEX idx_activity_logs_expires       ON activity_logs(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_activity_logs_cleanup       ON activity_logs(severity, expires_at, timestamp);
CREATE INDEX idx_activity_logs_user_timestamp ON activity_logs(user_id, timestamp DESC);
