-- Migration: Create email system tables (email_server_configs, email_types, email_templates)
-- and seed all default email types and templates
-- Depends on: 007_system_settings (applications table must exist)

-- ─── email_server_configs ─────────────────────────────────────────────────────

CREATE TABLE email_server_configs (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id        UUID         REFERENCES applications(id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL DEFAULT 'Default',
    smtp_host     VARCHAR(255) NOT NULL,
    smtp_port     INTEGER      NOT NULL DEFAULT 587,
    smtp_username VARCHAR(255)          DEFAULT '',
    smtp_password TEXT                  DEFAULT '',
    from_address  VARCHAR(255) NOT NULL,
    from_name     VARCHAR(100)          DEFAULT '',
    use_tls       BOOLEAN      NOT NULL DEFAULT TRUE,
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    is_default    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_server_configs_app_id              ON email_server_configs(app_id);
CREATE UNIQUE INDEX idx_email_server_configs_app_default  ON email_server_configs(app_id)  WHERE app_id IS NOT NULL AND is_default = TRUE;
CREATE UNIQUE INDEX idx_email_server_configs_global_default ON email_server_configs(is_default) WHERE app_id IS NULL AND is_default = TRUE;

-- ─── email_types ──────────────────────────────────────────────────────────────

CREATE TABLE email_types (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(50) NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    description     TEXT                  DEFAULT '',
    default_subject VARCHAR(255)          DEFAULT '',
    variables       JSONB                 DEFAULT '[]',
    is_system       BOOLEAN     NOT NULL DEFAULT TRUE,
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── email_templates ──────────────────────────────────────────────────────────

CREATE TABLE email_templates (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id          UUID        REFERENCES applications(id) ON DELETE CASCADE,
    email_type_id   UUID        NOT NULL REFERENCES email_types(id) ON DELETE CASCADE,
    server_config_id UUID       REFERENCES email_server_configs(id) ON DELETE SET NULL,
    name            VARCHAR(100) NOT NULL,
    subject         VARCHAR(255) NOT NULL,
    body_html       TEXT                  DEFAULT '',
    body_text       TEXT                  DEFAULT '',
    from_email      VARCHAR(255)          DEFAULT '',
    from_name       VARCHAR(255)          DEFAULT '',
    template_engine VARCHAR(20) NOT NULL DEFAULT 'go_template',
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_app_email_type    ON email_templates(app_id, email_type_id) WHERE app_id IS NOT NULL;
CREATE UNIQUE INDEX idx_global_email_type ON email_templates(email_type_id)         WHERE app_id IS NULL;

-- ============================================================
-- Seed email types (11 total)
-- ============================================================

-- First 6 email types
INSERT INTO email_types (code, name, description, default_subject, variables, is_system, is_active) VALUES
('email_verification', 'Email Verification', 'Sent when a user registers or changes their email address to verify ownership.', 'Verify Your Email Address',
'[{"name": "app_name", "description": "Application name", "required": true}, {"name": "user_email", "description": "User email address", "required": true}, {"name": "verification_link", "description": "Email verification URL", "required": true}, {"name": "verification_token", "description": "Raw verification token", "required": false}]'::jsonb, TRUE, TRUE),
('password_reset', 'Password Reset', 'Sent when a user requests a password reset link.', 'Reset Your Password',
'[{"name": "app_name", "description": "Application name", "required": true}, {"name": "user_email", "description": "User email address", "required": true}, {"name": "reset_link", "description": "Password reset URL", "required": true}, {"name": "expiration_minutes", "description": "Link expiration time in minutes", "required": false}]'::jsonb, TRUE, TRUE),
('two_fa_code', '2FA Verification Code', 'Sent when a user needs a 2FA verification code via email.', 'Your Verification Code',
'[{"name": "app_name", "description": "Application name", "required": true}, {"name": "user_email", "description": "User email address", "required": true}, {"name": "code", "description": "6-digit verification code", "required": true}, {"name": "expiration_minutes", "description": "Code expiration time in minutes", "required": false}]'::jsonb, TRUE, TRUE),
('welcome', 'Welcome Email', 'Sent to welcome a user after successful registration and email verification.', 'Welcome to {{.AppName}}',
'[{"name": "app_name", "description": "Application name", "required": true}, {"name": "user_email", "description": "User email address", "required": true}, {"name": "user_name", "description": "User display name", "required": false}]'::jsonb, TRUE, TRUE),
('account_deactivated', 'Account Deactivated', 'Sent when a user account is deactivated by an administrator.', 'Your Account Has Been Deactivated',
'[{"name": "app_name", "description": "Application name", "required": true}, {"name": "user_email", "description": "User email address", "required": true}, {"name": "user_name", "description": "User display name", "required": false}]'::jsonb, TRUE, TRUE),
('password_changed', 'Password Changed', 'Sent as a security notification when a user changes their password.', 'Your Password Has Been Changed',
'[{"name": "app_name", "description": "Application name", "required": true}, {"name": "user_email", "description": "User email address", "required": true}, {"name": "user_name", "description": "User display name", "required": false}, {"name": "change_time", "description": "Time of password change", "required": false}]'::jsonb, TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

-- magic_link
INSERT INTO email_types (code, name, description, default_subject, variables, is_system, is_active) VALUES
('magic_link', 'Magic Link Login', 'Sent when a user requests a passwordless login via email magic link.', 'Sign In to Your Account',
'[{"name": "app_name", "description": "Application name", "required": true}, {"name": "user_email", "description": "User email address", "required": true}, {"name": "magic_link", "description": "Magic link login URL", "required": true}, {"name": "expiration_minutes", "description": "Link expiration time in minutes", "required": false}]'::jsonb, TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

-- api_key_expiring_soon
INSERT INTO email_types (code, name, description, default_subject, variables, is_system, is_active) VALUES
('api_key_expiring_soon', 'API Key Expiring Soon', 'Sent to the system admin when an API key is approaching its expiration date (7-day and 1-day warnings).', 'API Key Expiring Soon',
'[{"name": "app_name", "description": "Application or system name", "required": true, "default_value": "Auth API"}, {"name": "api_key_name", "description": "Name/label of the expiring API key", "required": true}, {"name": "api_key_prefix", "description": "Key prefix identifier (safe to display)", "required": true}, {"name": "api_key_type", "description": "Type of key: admin or app", "required": true}, {"name": "api_key_expires_at", "description": "Formatted expiry date/time of the key", "required": true}, {"name": "days_until_expiry", "description": "Number of days until the key expires", "required": true}]'::jsonb, TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

-- new_device_login
INSERT INTO email_types (code, name, description, default_subject, variables, is_system, is_active) VALUES
('new_device_login', 'New Device Login Notification', 'Sent when a login is detected from a new device or location not previously seen for this account.', 'New Login to Your Account',
'[{"name": "app_name", "description": "Application name", "required": true}, {"name": "user_email", "description": "User email address", "required": true}, {"name": "login_ip", "description": "IP address of the login attempt", "required": true}, {"name": "login_location", "description": "Geographic location of the login (e.g. city, country)", "required": false}, {"name": "login_device", "description": "Device/browser user-agent of the login", "required": false}, {"name": "login_time", "description": "Timestamp of the login event", "required": true}]'::jsonb, TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

-- suspicious_activity
INSERT INTO email_types (code, name, description, default_subject, variables, is_system, is_active) VALUES
('suspicious_activity', 'Suspicious Activity Alert', 'Sent when suspicious activity is detected on a user account, such as brute-force attempts or unusual access patterns.', 'Security Alert: Suspicious Activity on Your Account',
'[{"name": "app_name", "description": "Application name", "required": true}, {"name": "user_email", "description": "User email address", "required": true}, {"name": "login_ip", "description": "IP address of the suspicious activity", "required": true}, {"name": "login_location", "description": "Geographic location of the activity (e.g. city, country)", "required": false}, {"name": "login_device", "description": "Device/browser user-agent", "required": false}, {"name": "login_time", "description": "Timestamp of the event", "required": true}, {"name": "alert_type", "description": "Type of security alert (e.g. new_device, brute_force)", "required": false}, {"name": "alert_details", "description": "Detailed description of the security alert", "required": true}]'::jsonb, TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

-- backup_email_verification
INSERT INTO email_types (code, name, description, default_subject, variables, is_system, is_active) VALUES
('backup_email_verification', 'Backup Email Verification', 'Sent when a user registers a backup email address for 2FA recovery. Contains a verification link the user must click to confirm ownership of the backup address.', 'Verify Your Backup Email Address',
'[{"name": "app_name", "description": "Application name", "required": true}, {"name": "backup_email", "description": "The backup email address being verified", "required": true}, {"name": "verification_link", "description": "Full URL the user must click to verify the backup address", "required": true}, {"name": "expiration_minutes", "description": "Number of minutes before the verification link expires", "required": false}]'::jsonb, TRUE, TRUE)
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- Seed default email templates (11 total, all global with app_id = NULL)
-- ============================================================

-- 1. Email Verification
INSERT INTO email_templates (app_id, email_type_id, name, subject, body_html, body_text, template_engine, is_active) VALUES
(
    NULL,
    (SELECT id FROM email_types WHERE code = 'email_verification'),
    'Default Email Verification',
    'Verify Your Email Address',
    '<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Verify Your Email</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fa;font-family:-apple-system,BlinkMacSystemFont,''Segoe UI'',Roboto,''Helvetica Neue'',Arial,sans-serif;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f4f7fa;padding:40px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellspacing="0" cellpadding="0" style="background-color:#ffffff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
  <tr><td style="background-color:#4f46e5;padding:32px 40px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;font-weight:600;">{{.AppName}}</h1>
  </td></tr>
  <tr><td style="padding:40px;">
    <h2 style="color:#1a1a2e;margin:0 0 16px;font-size:20px;">Verify Your Email Address</h2>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      Thank you for registering. Please click the button below to verify your email address and activate your account.
    </p>
    <table role="presentation" cellspacing="0" cellpadding="0" style="margin:0 auto 24px;">
    <tr><td style="background-color:#4f46e5;border-radius:6px;">
      <a href="{{.VerificationLink}}" style="display:inline-block;padding:14px 32px;color:#ffffff;text-decoration:none;font-size:16px;font-weight:600;">Verify Email Address</a>
    </td></tr>
    </table>
    <p style="color:#718096;font-size:14px;line-height:1.5;margin:0 0 8px;">
      If the button doesn''t work, copy and paste this link into your browser:
    </p>
    <p style="color:#4f46e5;font-size:14px;word-break:break-all;margin:0 0 24px;">{{.VerificationLink}}</p>
    <p style="color:#a0aec0;font-size:13px;margin:0;">
      If you did not create an account, you can safely ignore this email.
    </p>
  </td></tr>
  <tr><td style="background-color:#f8fafc;padding:24px 40px;text-align:center;border-top:1px solid #e2e8f0;">
    <p style="color:#a0aec0;font-size:12px;margin:0;">This email was sent by {{.AppName}}. Please do not reply to this email.</p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>',
    'Verify Your Email Address

Thank you for registering with {{.AppName}}.

Please verify your email address by clicking the link below:
{{.VerificationLink}}

If you did not create an account, you can safely ignore this email.',
    'go_template',
    TRUE
)
ON CONFLICT (email_type_id) WHERE app_id IS NULL DO NOTHING;

-- 2. Password Reset
INSERT INTO email_templates (app_id, email_type_id, name, subject, body_html, body_text, template_engine, is_active) VALUES
(
    NULL,
    (SELECT id FROM email_types WHERE code = 'password_reset'),
    'Default Password Reset',
    'Reset Your Password',
    '<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Reset Your Password</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fa;font-family:-apple-system,BlinkMacSystemFont,''Segoe UI'',Roboto,''Helvetica Neue'',Arial,sans-serif;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f4f7fa;padding:40px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellspacing="0" cellpadding="0" style="background-color:#ffffff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
  <tr><td style="background-color:#4f46e5;padding:32px 40px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;font-weight:600;">{{.AppName}}</h1>
  </td></tr>
  <tr><td style="padding:40px;">
    <h2 style="color:#1a1a2e;margin:0 0 16px;font-size:20px;">Reset Your Password</h2>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      We received a request to reset your password. Click the button below to choose a new password.
    </p>
    <table role="presentation" cellspacing="0" cellpadding="0" style="margin:0 auto 24px;">
    <tr><td style="background-color:#4f46e5;border-radius:6px;">
      <a href="{{.ResetLink}}" style="display:inline-block;padding:14px 32px;color:#ffffff;text-decoration:none;font-size:16px;font-weight:600;">Reset Password</a>
    </td></tr>
    </table>
    <p style="color:#718096;font-size:14px;line-height:1.5;margin:0 0 8px;">
      If the button doesn''t work, copy and paste this link into your browser:
    </p>
    <p style="color:#4f46e5;font-size:14px;word-break:break-all;margin:0 0 24px;">{{.ResetLink}}</p>
    <p style="color:#e53e3e;font-size:14px;line-height:1.5;margin:0 0 16px;">
      This link will expire in {{.ExpirationMinutes}} minutes.
    </p>
    <p style="color:#a0aec0;font-size:13px;margin:0;">
      If you didn''t request a password reset, you can safely ignore this email. Your password will not be changed.
    </p>
  </td></tr>
  <tr><td style="background-color:#f8fafc;padding:24px 40px;text-align:center;border-top:1px solid #e2e8f0;">
    <p style="color:#a0aec0;font-size:12px;margin:0;">This email was sent by {{.AppName}}. Please do not reply to this email.</p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>',
    'Reset Your Password

We received a request to reset your password for your {{.AppName}} account.

Click the link below to reset your password:
{{.ResetLink}}

This link will expire in {{.ExpirationMinutes}} minutes.

If you didn''t request a password reset, you can safely ignore this email.',
    'go_template',
    TRUE
)
ON CONFLICT (email_type_id) WHERE app_id IS NULL DO NOTHING;

-- 3. Two-Factor Authentication Code
INSERT INTO email_templates (app_id, email_type_id, name, subject, body_html, body_text, template_engine, is_active) VALUES
(
    NULL,
    (SELECT id FROM email_types WHERE code = 'two_fa_code'),
    'Default 2FA Verification Code',
    'Your Verification Code',
    '<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Your Verification Code</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fa;font-family:-apple-system,BlinkMacSystemFont,''Segoe UI'',Roboto,''Helvetica Neue'',Arial,sans-serif;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f4f7fa;padding:40px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellspacing="0" cellpadding="0" style="background-color:#ffffff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
  <tr><td style="background-color:#4f46e5;padding:32px 40px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;font-weight:600;">{{.AppName}}</h1>
  </td></tr>
  <tr><td style="padding:40px;text-align:center;">
    <h2 style="color:#1a1a2e;margin:0 0 16px;font-size:20px;">Your Verification Code</h2>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 32px;">
      Use the following code to complete your sign-in. This code is valid for {{.ExpirationMinutes}} minutes.
    </p>
    <div style="background-color:#f0f4ff;border:2px solid #4f46e5;border-radius:12px;padding:24px;display:inline-block;margin:0 0 32px;">
      <span style="font-size:36px;font-weight:700;letter-spacing:8px;color:#1a1a2e;font-family:''Courier New'',monospace;">{{.Code}}</span>
    </div>
    <p style="color:#e53e3e;font-size:14px;line-height:1.5;margin:0 0 16px;">
      Do not share this code with anyone. Our team will never ask for your code.
    </p>
    <p style="color:#a0aec0;font-size:13px;margin:0;">
      If you did not request this code, someone may be trying to access your account. Please change your password immediately.
    </p>
  </td></tr>
  <tr><td style="background-color:#f8fafc;padding:24px 40px;text-align:center;border-top:1px solid #e2e8f0;">
    <p style="color:#a0aec0;font-size:12px;margin:0;">This email was sent by {{.AppName}}. Please do not reply to this email.</p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>',
    'Your Verification Code

Use the following code to complete your sign-in to {{.AppName}}:

{{.Code}}

This code is valid for {{.ExpirationMinutes}} minutes.

Do not share this code with anyone. Our team will never ask for your code.

If you did not request this code, please change your password immediately.',
    'go_template',
    TRUE
)
ON CONFLICT (email_type_id) WHERE app_id IS NULL DO NOTHING;

-- 4. Welcome Email
INSERT INTO email_templates (app_id, email_type_id, name, subject, body_html, body_text, template_engine, is_active) VALUES
(
    NULL,
    (SELECT id FROM email_types WHERE code = 'welcome'),
    'Default Welcome Email',
    'Welcome to {{.AppName}}',
    '<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Welcome</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fa;font-family:-apple-system,BlinkMacSystemFont,''Segoe UI'',Roboto,''Helvetica Neue'',Arial,sans-serif;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f4f7fa;padding:40px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellspacing="0" cellpadding="0" style="background-color:#ffffff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
  <tr><td style="background-color:#4f46e5;padding:32px 40px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;font-weight:600;">{{.AppName}}</h1>
  </td></tr>
  <tr><td style="padding:40px;">
    <h2 style="color:#1a1a2e;margin:0 0 16px;font-size:20px;">Welcome!</h2>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      Your email has been verified and your account is now active. Welcome to {{.AppName}}!
    </p>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      Here are a few things you can do to get started:
    </p>
    <ul style="color:#4a5568;font-size:16px;line-height:1.8;margin:0 0 24px;padding-left:24px;">
      <li>Complete your profile information</li>
      <li>Set up two-factor authentication for added security</li>
      <li>Explore the features available to you</li>
    </ul>
    <p style="color:#a0aec0;font-size:13px;margin:0;">
      If you have any questions, please contact our support team.
    </p>
  </td></tr>
  <tr><td style="background-color:#f8fafc;padding:24px 40px;text-align:center;border-top:1px solid #e2e8f0;">
    <p style="color:#a0aec0;font-size:12px;margin:0;">This email was sent by {{.AppName}}. Please do not reply to this email.</p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>',
    'Welcome to {{.AppName}}!

Your email has been verified and your account is now active.

Here are a few things you can do to get started:
- Complete your profile information
- Set up two-factor authentication for added security
- Explore the features available to you

If you have any questions, please contact our support team.',
    'go_template',
    TRUE
)
ON CONFLICT (email_type_id) WHERE app_id IS NULL DO NOTHING;

-- 5. Account Deactivated
INSERT INTO email_templates (app_id, email_type_id, name, subject, body_html, body_text, template_engine, is_active) VALUES
(
    NULL,
    (SELECT id FROM email_types WHERE code = 'account_deactivated'),
    'Default Account Deactivated',
    'Your Account Has Been Deactivated',
    '<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Account Deactivated</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fa;font-family:-apple-system,BlinkMacSystemFont,''Segoe UI'',Roboto,''Helvetica Neue'',Arial,sans-serif;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f4f7fa;padding:40px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellspacing="0" cellpadding="0" style="background-color:#ffffff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
  <tr><td style="background-color:#e53e3e;padding:32px 40px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;font-weight:600;">{{.AppName}}</h1>
  </td></tr>
  <tr><td style="padding:40px;">
    <h2 style="color:#1a1a2e;margin:0 0 16px;font-size:20px;">Account Deactivated</h2>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      Your account on {{.AppName}} has been deactivated. You will no longer be able to sign in or access your account.
    </p>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      If you believe this was done in error, please contact the application administrator to have your account reactivated.
    </p>
    <p style="color:#a0aec0;font-size:13px;margin:0;">
      This is an automated notification. Please do not reply to this email.
    </p>
  </td></tr>
  <tr><td style="background-color:#f8fafc;padding:24px 40px;text-align:center;border-top:1px solid #e2e8f0;">
    <p style="color:#a0aec0;font-size:12px;margin:0;">This email was sent by {{.AppName}}.</p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>',
    'Account Deactivated

Your account on {{.AppName}} has been deactivated. You will no longer be able to sign in or access your account.

If you believe this was done in error, please contact the application administrator to have your account reactivated.',
    'go_template',
    TRUE
)
ON CONFLICT (email_type_id) WHERE app_id IS NULL DO NOTHING;

-- 6. Password Changed
INSERT INTO email_templates (app_id, email_type_id, name, subject, body_html, body_text, template_engine, is_active) VALUES
(
    NULL,
    (SELECT id FROM email_types WHERE code = 'password_changed'),
    'Default Password Changed',
    'Your Password Has Been Changed',
    '<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Password Changed</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fa;font-family:-apple-system,BlinkMacSystemFont,''Segoe UI'',Roboto,''Helvetica Neue'',Arial,sans-serif;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f4f7fa;padding:40px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellspacing="0" cellpadding="0" style="background-color:#ffffff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
  <tr><td style="background-color:#f6ad55;padding:32px 40px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;font-weight:600;">{{.AppName}}</h1>
  </td></tr>
  <tr><td style="padding:40px;">
    <h2 style="color:#1a1a2e;margin:0 0 16px;font-size:20px;">Password Changed</h2>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      Your password for {{.AppName}} was successfully changed on {{.ChangeTime}}.
    </p>
    <p style="color:#e53e3e;font-size:16px;line-height:1.6;margin:0 0 24px;font-weight:600;">
      If you did not make this change, please reset your password immediately and contact support.
    </p>
    <p style="color:#a0aec0;font-size:13px;margin:0;">
      This is a security notification. Please do not reply to this email.
    </p>
  </td></tr>
  <tr><td style="background-color:#f8fafc;padding:24px 40px;text-align:center;border-top:1px solid #e2e8f0;">
    <p style="color:#a0aec0;font-size:12px;margin:0;">This email was sent by {{.AppName}}.</p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>',
    'Password Changed

Your password for {{.AppName}} was successfully changed on {{.ChangeTime}}.

If you did not make this change, please reset your password immediately and contact support.

This is a security notification.',
    'go_template',
    TRUE
)
ON CONFLICT (email_type_id) WHERE app_id IS NULL DO NOTHING;

-- 7. Magic Link Login
INSERT INTO email_templates (app_id, email_type_id, name, subject, body_html, body_text, template_engine, is_active) VALUES
(
    NULL,
    (SELECT id FROM email_types WHERE code = 'magic_link'),
    'Default Magic Link Login',
    'Sign In to Your Account',
    '<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Sign In to Your Account</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fa;font-family:-apple-system,BlinkMacSystemFont,''Segoe UI'',Roboto,''Helvetica Neue'',Arial,sans-serif;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f4f7fa;padding:40px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellspacing="0" cellpadding="0" style="background-color:#ffffff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
  <tr><td style="background-color:#4f46e5;padding:32px 40px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;font-weight:600;">{{.AppName}}</h1>
  </td></tr>
  <tr><td style="padding:40px;">
    <h2 style="color:#1a1a2e;margin:0 0 16px;font-size:20px;">Sign In to Your Account</h2>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      We received a request to sign in to your account. Click the button below to log in instantly — no password needed.
    </p>
    <table role="presentation" cellspacing="0" cellpadding="0" style="margin:0 auto 24px;">
    <tr><td style="background-color:#4f46e5;border-radius:6px;">
      <a href="{{.MagicLink}}" style="display:inline-block;padding:14px 32px;color:#ffffff;text-decoration:none;font-size:16px;font-weight:600;">Sign In Now</a>
    </td></tr>
    </table>
    <p style="color:#718096;font-size:14px;line-height:1.5;margin:0 0 8px;">
      If the button doesn''t work, copy and paste this link into your browser:
    </p>
    <p style="color:#4f46e5;font-size:14px;word-break:break-all;margin:0 0 24px;">{{.MagicLink}}</p>
    <p style="color:#e53e3e;font-size:14px;line-height:1.5;margin:0 0 16px;">
      This link will expire in {{.ExpirationMinutes}} minutes and can only be used once.
    </p>
    <p style="color:#a0aec0;font-size:13px;margin:0;">
      If you didn''t request this link, you can safely ignore this email. No one can access your account without clicking the link above.
    </p>
  </td></tr>
  <tr><td style="background-color:#f8fafc;padding:24px 40px;text-align:center;border-top:1px solid #e2e8f0;">
    <p style="color:#a0aec0;font-size:12px;margin:0;">This email was sent by {{.AppName}}. Please do not reply to this email.</p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>',
    'Sign In to Your Account

We received a request to sign in to your {{.AppName}} account.

Click the link below to log in instantly:
{{.MagicLink}}

This link will expire in {{.ExpirationMinutes}} minutes and can only be used once.

If you didn''t request this link, you can safely ignore this email.',
    'go_template',
    TRUE
)
ON CONFLICT (email_type_id) WHERE app_id IS NULL DO NOTHING;

-- 8. API Key Expiring Soon
INSERT INTO email_templates (app_id, email_type_id, name, subject, body_html, body_text, template_engine, is_active) VALUES
(
    NULL,
    (SELECT id FROM email_types WHERE code = 'api_key_expiring_soon'),
    'Default API Key Expiring Soon',
    'API Key ''{{.ApiKeyName}}'' expires in {{.DaysUntilExpiry}} days',
    '<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>API Key Expiring Soon</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fa;font-family:-apple-system,BlinkMacSystemFont,''Segoe UI'',Roboto,''Helvetica Neue'',Arial,sans-serif;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f4f7fa;padding:40px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellspacing="0" cellpadding="0" style="background-color:#ffffff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
  <tr><td style="background-color:#4f46e5;padding:32px 40px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;font-weight:600;">{{.AppName}}</h1>
  </td></tr>
  <tr><td style="padding:40px;">
    <h2 style="color:#1a1a2e;margin:0 0 16px;font-size:20px;">
      <span style="color:#d97706;">&#9888;</span> API Key Expiring Soon
    </h2>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      An API key in your <strong>{{.AppName}}</strong> account will expire in
      <strong>{{.DaysUntilExpiry}} day{{if ne .DaysUntilExpiry "1"}}s{{end}}</strong>.
      Please rotate this key before it expires to avoid service interruption.
    </p>
    <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f8fafc;border-radius:6px;padding:20px;margin:0 0 24px;">
      <tr>
        <td style="padding:6px 0;color:#64748b;font-size:14px;width:140px;">Key Name:</td>
        <td style="padding:6px 0;color:#1e293b;font-size:14px;font-weight:600;">{{.ApiKeyName}}</td>
      </tr>
      <tr>
        <td style="padding:6px 0;color:#64748b;font-size:14px;">Key Identifier:</td>
        <td style="padding:6px 0;color:#1e293b;font-size:14px;font-family:monospace;">{{.ApiKeyPrefix}}...</td>
      </tr>
      <tr>
        <td style="padding:6px 0;color:#64748b;font-size:14px;">Key Type:</td>
        <td style="padding:6px 0;color:#1e293b;font-size:14px;text-transform:capitalize;">{{.ApiKeyType}}</td>
      </tr>
      <tr>
        <td style="padding:6px 0;color:#64748b;font-size:14px;">Expires At:</td>
        <td style="padding:6px 0;color:#dc2626;font-size:14px;font-weight:600;">{{.ApiKeyExpiresAt}}</td>
      </tr>
    </table>
    <p style="color:#4a5568;font-size:15px;line-height:1.6;margin:0 0 24px;">
      To rotate this key, log in to the admin panel and generate a new API key, then update your integrations before the expiry date.
    </p>
    <p style="color:#9ca3af;font-size:13px;line-height:1.6;margin:0;">
      This is an automated notification from {{.AppName}}. If you did not expect this email or have already rotated this key, you can safely ignore it.
    </p>
  </td></tr>
  <tr><td style="background-color:#f8fafc;padding:20px 40px;text-align:center;border-top:1px solid #e2e8f0;">
    <p style="color:#94a3b8;font-size:12px;margin:0;">
      &copy; {{.AppName}} &mdash; Automated Security Notification
    </p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>',
    'API Key Expiring Soon

An API key in your {{.AppName}} account will expire in {{.DaysUntilExpiry}} day(s).

Key Name:       {{.ApiKeyName}}
Key Identifier: {{.ApiKeyPrefix}}...
Key Type:       {{.ApiKeyType}}
Expires At:     {{.ApiKeyExpiresAt}}

Please rotate this key before it expires to avoid service interruption.
Log in to the admin panel and generate a new API key, then update your integrations.

This is an automated notification from {{.AppName}}.',
    'go_template',
    TRUE
)
ON CONFLICT (email_type_id) WHERE app_id IS NULL DO NOTHING;

-- 9. New Device Login Notification
INSERT INTO email_templates (app_id, email_type_id, name, subject, body_html, body_text, template_engine, is_active) VALUES
(
    NULL,
    (SELECT id FROM email_types WHERE code = 'new_device_login'),
    'Default New Device Login Notification',
    'New Login to Your Account',
    '<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>New Login Detected</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fa;font-family:-apple-system,BlinkMacSystemFont,''Segoe UI'',Roboto,''Helvetica Neue'',Arial,sans-serif;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f4f7fa;padding:40px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellspacing="0" cellpadding="0" style="background-color:#ffffff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
  <tr><td style="background-color:#3182ce;padding:32px 40px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;font-weight:600;">{{.AppName}}</h1>
  </td></tr>
  <tr><td style="padding:40px;">
    <h2 style="color:#1a1a2e;margin:0 0 16px;font-size:20px;">New Login Detected</h2>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      We noticed a new login to your account from a device or location we haven''t seen before:
    </p>
    <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f7fafc;border-radius:8px;border:1px solid #e2e8f0;margin:0 0 24px;">
      <tr><td style="padding:20px;">
        <table role="presentation" width="100%" cellspacing="0" cellpadding="4">
          <tr>
            <td style="color:#718096;font-size:14px;width:100px;vertical-align:top;padding:4px 8px;">IP Address:</td>
            <td style="color:#1a1a2e;font-size:14px;font-weight:600;padding:4px 8px;">{{.LoginIP}}</td>
          </tr>
          <tr>
            <td style="color:#718096;font-size:14px;vertical-align:top;padding:4px 8px;">Location:</td>
            <td style="color:#1a1a2e;font-size:14px;font-weight:600;padding:4px 8px;">{{.LoginLocation}}</td>
          </tr>
          <tr>
            <td style="color:#718096;font-size:14px;vertical-align:top;padding:4px 8px;">Device:</td>
            <td style="color:#1a1a2e;font-size:14px;padding:4px 8px;">{{.LoginDevice}}</td>
          </tr>
          <tr>
            <td style="color:#718096;font-size:14px;vertical-align:top;padding:4px 8px;">Time:</td>
            <td style="color:#1a1a2e;font-size:14px;font-weight:600;padding:4px 8px;">{{.LoginTime}}</td>
          </tr>
        </table>
      </td></tr>
    </table>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 16px;">
      If this was you, you can ignore this email. If you don''t recognize this activity, we recommend changing your password immediately.
    </p>
    <p style="color:#a0aec0;font-size:13px;margin:0;">
      This is an automated security notification from {{.AppName}}.
    </p>
  </td></tr>
  <tr><td style="background-color:#f8fafc;padding:24px 40px;text-align:center;border-top:1px solid #e2e8f0;">
    <p style="color:#a0aec0;font-size:12px;margin:0;">This email was sent by {{.AppName}}. Please do not reply to this email.</p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>',
    'New Login Detected

We noticed a new login to your {{.AppName}} account:

IP Address: {{.LoginIP}}
Location:   {{.LoginLocation}}
Device:     {{.LoginDevice}}
Time:       {{.LoginTime}}

If this was you, you can ignore this email. If you don''t recognize this activity, we recommend changing your password immediately.

This is an automated security notification from {{.AppName}}.',
    'go_template',
    TRUE
)
ON CONFLICT (email_type_id) WHERE app_id IS NULL DO NOTHING;

-- 10. Suspicious Activity Alert
INSERT INTO email_templates (app_id, email_type_id, name, subject, body_html, body_text, template_engine, is_active) VALUES
(
    NULL,
    (SELECT id FROM email_types WHERE code = 'suspicious_activity'),
    'Default Suspicious Activity Alert',
    'Security Alert: Suspicious Activity on Your Account',
    '<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Security Alert</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fa;font-family:-apple-system,BlinkMacSystemFont,''Segoe UI'',Roboto,''Helvetica Neue'',Arial,sans-serif;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f4f7fa;padding:40px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellspacing="0" cellpadding="0" style="background-color:#ffffff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
  <tr><td style="background-color:#e53e3e;padding:32px 40px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;font-weight:600;">{{.AppName}}</h1>
  </td></tr>
  <tr><td style="padding:40px;">
    <h2 style="color:#1a1a2e;margin:0 0 16px;font-size:20px;">Security Alert</h2>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      We detected suspicious activity on your account. Please review the details below:
    </p>
    <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#fff5f5;border-radius:8px;border:1px solid #feb2b2;margin:0 0 24px;">
      <tr><td style="padding:20px;">
        <table role="presentation" width="100%" cellspacing="0" cellpadding="4">
          <tr>
            <td style="color:#718096;font-size:14px;width:100px;vertical-align:top;padding:4px 8px;">Alert:</td>
            <td style="color:#e53e3e;font-size:14px;font-weight:600;padding:4px 8px;">{{.AlertDetails}}</td>
          </tr>
          <tr>
            <td style="color:#718096;font-size:14px;vertical-align:top;padding:4px 8px;">IP Address:</td>
            <td style="color:#1a1a2e;font-size:14px;font-weight:600;padding:4px 8px;">{{.LoginIP}}</td>
          </tr>
          <tr>
            <td style="color:#718096;font-size:14px;vertical-align:top;padding:4px 8px;">Location:</td>
            <td style="color:#1a1a2e;font-size:14px;font-weight:600;padding:4px 8px;">{{.LoginLocation}}</td>
          </tr>
          <tr>
            <td style="color:#718096;font-size:14px;vertical-align:top;padding:4px 8px;">Device:</td>
            <td style="color:#1a1a2e;font-size:14px;padding:4px 8px;">{{.LoginDevice}}</td>
          </tr>
          <tr>
            <td style="color:#718096;font-size:14px;vertical-align:top;padding:4px 8px;">Time:</td>
            <td style="color:#1a1a2e;font-size:14px;font-weight:600;padding:4px 8px;">{{.LoginTime}}</td>
          </tr>
        </table>
      </td></tr>
    </table>
    <p style="color:#e53e3e;font-size:16px;line-height:1.6;margin:0 0 16px;font-weight:600;">
      If you don''t recognize this activity, we strongly recommend:
    </p>
    <ul style="color:#4a5568;font-size:16px;line-height:1.8;margin:0 0 24px;padding-left:24px;">
      <li>Changing your password immediately</li>
      <li>Enabling two-factor authentication if not already active</li>
      <li>Reviewing your recent account activity</li>
    </ul>
    <p style="color:#a0aec0;font-size:13px;margin:0;">
      This is an automated security alert from {{.AppName}}.
    </p>
  </td></tr>
  <tr><td style="background-color:#f8fafc;padding:24px 40px;text-align:center;border-top:1px solid #e2e8f0;">
    <p style="color:#a0aec0;font-size:12px;margin:0;">This email was sent by {{.AppName}}. Please do not reply to this email.</p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>',
    'Security Alert: Suspicious Activity on Your Account

We detected suspicious activity on your {{.AppName}} account:

Alert:      {{.AlertDetails}}
IP Address: {{.LoginIP}}
Location:   {{.LoginLocation}}
Device:     {{.LoginDevice}}
Time:       {{.LoginTime}}

If you don''t recognize this activity, we strongly recommend:
- Changing your password immediately
- Enabling two-factor authentication if not already active
- Reviewing your recent account activity

This is an automated security alert from {{.AppName}}.',
    'go_template',
    TRUE
)
ON CONFLICT (email_type_id) WHERE app_id IS NULL DO NOTHING;

-- 11. Backup Email Verification
INSERT INTO email_templates (app_id, email_type_id, name, subject, body_html, body_text, template_engine, is_active) VALUES
(
    NULL,
    (SELECT id FROM email_types WHERE code = 'backup_email_verification'),
    'Default Backup Email Verification',
    'Verify Your Backup Email Address',
    '<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Verify Your Backup Email</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fa;font-family:-apple-system,BlinkMacSystemFont,''Segoe UI'',Roboto,''Helvetica Neue'',Arial,sans-serif;">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#f4f7fa;padding:40px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellspacing="0" cellpadding="0" style="background-color:#ffffff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.08);overflow:hidden;">
  <tr><td style="background-color:#4f46e5;padding:32px 40px;text-align:center;">
    <h1 style="color:#ffffff;margin:0;font-size:24px;font-weight:600;">{{.AppName}}</h1>
  </td></tr>
  <tr><td style="padding:40px;">
    <h2 style="color:#1a1a2e;margin:0 0 16px;font-size:20px;">Verify Your Backup Email Address</h2>
    <p style="color:#4a5568;font-size:16px;line-height:1.6;margin:0 0 24px;">
      You requested to register <strong>{{.BackupEmail}}</strong> as a backup email address for account recovery.
      Please click the button below to verify this email address.
    </p>
    <table role="presentation" cellspacing="0" cellpadding="0" style="margin:0 auto 24px;">
    <tr><td style="background-color:#4f46e5;border-radius:6px;">
      <a href="{{.VerificationLink}}" style="display:inline-block;padding:14px 32px;color:#ffffff;text-decoration:none;font-size:16px;font-weight:600;">Verify Backup Email</a>
    </td></tr>
    </table>
    <p style="color:#718096;font-size:14px;line-height:1.5;margin:0 0 8px;">
      If the button doesn''t work, copy and paste this link into your browser:
    </p>
    <p style="color:#4f46e5;font-size:14px;word-break:break-all;margin:0 0 24px;">{{.VerificationLink}}</p>
    <p style="color:#e53e3e;font-size:14px;line-height:1.5;margin:0 0 16px;">
      This link will expire in {{.ExpirationMinutes}} minutes.
    </p>
    <p style="color:#a0aec0;font-size:13px;margin:0;">
      If you did not request this, you can safely ignore this email. Your primary account will not be affected.
    </p>
  </td></tr>
  <tr><td style="background-color:#f8fafc;padding:24px 40px;text-align:center;border-top:1px solid #e2e8f0;">
    <p style="color:#a0aec0;font-size:12px;margin:0;">This email was sent by {{.AppName}}. Please do not reply to this email.</p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>',
    'Verify Your Backup Email Address

You requested to register {{.BackupEmail}} as a backup email address for your {{.AppName}} account recovery.

Please verify this email address by clicking the link below:
{{.VerificationLink}}

This link will expire in {{.ExpirationMinutes}} minutes.

If you did not request this, you can safely ignore this email.',
    'go_template',
    TRUE
)
ON CONFLICT (email_type_id) WHERE app_id IS NULL DO NOTHING;
