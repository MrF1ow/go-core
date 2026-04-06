-- ─── webauthn_credentials ─────────────────────────────────────────────────────

CREATE TABLE webauthn_credentials (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID         REFERENCES users(id)         ON DELETE CASCADE,
    app_id           UUID         REFERENCES applications(id)  ON DELETE CASCADE,
    admin_id         UUID         REFERENCES admin_accounts(id) ON DELETE CASCADE,
    credential_id    BYTEA        NOT NULL,
    public_key       BYTEA        NOT NULL,
    attestation_type VARCHAR(50),
    aaguid           BYTEA,
    sign_count       INTEGER      NOT NULL DEFAULT 0,
    name             VARCHAR(100),
    transports       VARCHAR(255),
    backup_eligible  BOOLEAN      NOT NULL DEFAULT FALSE,
    backup_state     BOOLEAN      NOT NULL DEFAULT FALSE,
    last_used_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_webauthn_credentials_credential_id ON webauthn_credentials(credential_id);
CREATE INDEX idx_webauthn_credentials_user_id              ON webauthn_credentials(user_id);
CREATE INDEX idx_webauthn_credentials_app_id               ON webauthn_credentials(app_id);
CREATE INDEX idx_webauthn_credentials_admin_id             ON webauthn_credentials(admin_id);
