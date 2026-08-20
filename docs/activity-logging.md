# Activity logging

Security events go to `activity_logs`. Critical and important events are always written. High-frequency informational events (token refresh, profile GET) are off unless you enable them or anomaly detection fires.

This is not part of `core.Config`. Defaults come from `internal/config/logging.go` (environment), then the admin Settings page can override the same keys in the database.

## Severity

| Severity | Examples | Default retention |
|----------|----------|-------------------|
| Critical | LOGIN, LOGOUT, PASSWORD_CHANGE, 2FA enable/disable, ACCOUNT_LOCKED / UNLOCKED, ACCOUNT_DELETION | 365 days |
| Important | EMAIL_VERIFY, SOCIAL_LOGIN, PROFILE_UPDATE, trusted-device changes | 180 days |
| Informational | TOKEN_REFRESH, PROFILE_ACCESS, passkey and magic-link noise, BRUTE_FORCE_ATTEMPT | 90 days |

REGISTER is treated as critical in code. Informational events still get written when anomaly detection sees a new IP or user agent (default on, 30-day window).

## Defaults

- Token refresh and profile access: not logged
- Anomaly detection: on (new IP, new user agent)
- Cleanup: on, every 24 hours, 1000-row batches

## Settings keys

These work as env vars for the reference process and as GUI settings:

```bash
LOG_TOKEN_REFRESH=false
LOG_PROFILE_ACCESS=false
LOG_ANOMALY_DETECTION_ENABLED=true
LOG_ANOMALY_NEW_IP=true
LOG_ANOMALY_NEW_USER_AGENT=true
LOG_RETENTION_CRITICAL=365
LOG_RETENTION_IMPORTANT=180
LOG_RETENTION_INFORMATIONAL=90
LOG_CLEANUP_ENABLED=true
LOG_CLEANUP_INTERVAL=24h
```

Sampling (`LOG_SAMPLE_TOKEN_REFRESH`, `LOG_SAMPLE_PROFILE_ACCESS`) only applies when those events are enabled. Full list: [Environment variables](guides/ENV_VARIABLES.md).

## Export

- `GET /activity-logs/export`: the caller's logs (JWT)
- `GET /admin/activity-logs/export`: all logs (`X-Admin-API-Key`)

Query filters match the list endpoints (dates, event type, severity).
