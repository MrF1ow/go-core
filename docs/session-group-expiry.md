# Session group expiry

When a refresh-token session hits its Redis TTL, the expiry service can revoke that user's sessions in every other app in the same session group. Explicit logout already did this. Expiry is the same path, triggered by Redis instead of `POST /logout`.

It only runs if the group has `GlobalLogout` enabled and `Config.Session.GroupExpiryEnabled` is true.

## Config

```go
cfg.Session = core.SessionConfig{
    GroupExpiryEnabled:        true,
    GroupExpiryScanInterval:   5 * time.Minute,
    GroupKeyspaceNotifEnabled: true,
    RedisNotifyKeyspaceEvents: "Ex",
}
```

Reference app env:

```bash
SESSION_GROUP_EXPIRY_REVOCATION_ENABLED=true
SESSION_GROUP_EXPIRY_SCAN_INTERVAL=5m
SESSION_GROUP_KEYSPACE_NOTIF_ENABLED=true
REDIS_NOTIFY_KEYSPACE_EVENTS=Ex
```

The bundled Redis containers already start with `--notify-keyspace-events Ex`. For an external Redis, set that in `redis.conf`.

Wired in `internal/coreapp` via `sessiongroup.NewExpiryService`. Logout and OIDC end-session share `sessiongroup.Revoker`.

## How a session is watched

On login, Redis stores:

- Session hash `app:{appID}:session:{sessionID}`
- Metadata `session_meta:{appID}:{userID}:{sessionID}` with the same TTL

When the metadata key expires, Redis publishes `__keyevent@<db>__:expired`. The listener is currently subscribed to `__keyevent@0__:expired`. `DefaultConfig()` and the reference app use `REDIS_DB=1`, so real-time expiry does not fire on the documented Redis DB. The SCAN fallback every `GroupExpiryScanInterval` still covers missed events.

If keyspace notifications are off, a SCAN every `GroupExpiryScanInterval` looks for `session_meta:*` keys with TTL ≤ 0.

Manual revoke deletes the metadata key so expiry does not fire twice.

## What to expect

| Situation | Result |
|-----------|--------|
| User in apps A, B, C; group GlobalLogout on; A TTL hits 0 | B and C sessions revoked |
| User logs out of A | Same revocation (existing logout path) |
| App not in a group | Only that app's session dies |
| Group exists, GlobalLogout off | Only the expired app |

Admin session revoke ignores `GlobalLogout` and always tears everything down.

Refreshing a token resets both TTLs, so a healthy session does not look expired.
