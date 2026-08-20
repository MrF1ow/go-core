# Multi-tenancy

A tenant owns applications. Each application has its own users, OAuth providers, logs, and settings.

```
Tenant
 └── Application
      ├── Users
      ├── OAuth providers
      └── Activity logs
```

`Config.MultiTenant` defaults to `false`. In that mode a missing `X-App-ID` is filled with the default app. Set it to `true` when more than one app shares the process.

## Default IDs

Created by `migrations/001_tenants_and_apps.sql`:

- Tenant `00000000-0000-0000-0000-000000000001`
- Application `00000000-0000-0000-0000-000000000001`

## `X-App-ID`

Required when `MultiTenant` is true. Accepted on the header or `?app_id=`. Not required for `/swagger`, `/admin`, the admin GUI path, `/oidc`, or OAuth callbacks (app ID is in OAuth state).

```bash
curl -X POST http://localhost:8080/register \
  -H "X-App-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123!@#"}'
```

## Admin API

Admin JSON routes use `X-Admin-API-Key` (`Config.Admin.APIKey` or a key created in the GUI). They do not take a user JWT.

Create a tenant and app:

```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "X-Admin-API-Key: $ADMIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "Acme Corporation"}'

curl -X POST http://localhost:8080/admin/apps \
  -H "X-Admin-API-Key: $ADMIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Mobile App"
  }'
```

OAuth client credentials are per app:

```bash
curl -X POST http://localhost:8080/admin/oauth-providers \
  -H "X-Admin-API-Key: $ADMIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "660e8400-e29b-41d4-a716-446655440000",
    "provider": "google",
    "client_id": "....apps.googleusercontent.com",
    "client_secret": "...",
    "redirect_url": "https://mobile.example.com/auth/google/callback",
    "is_enabled": true
  }'
```

If you still have provider secrets in an old `.env`, configure them per app in the admin GUI (OAuth Configs) or `POST /admin/oauth-providers`.

## Isolation

- Unique email is `(email, app_id)`, not global
- JWTs carry `app_id`; a token from app A is rejected on app B
- Social accounts, 2FA secrets, and logs are per app
- Deleting an application cascades its users
