# Phase 2: Access-log client HTTP

[Overview](overview.md)

PR: Access client. Same PR as [phase 1](phase-1-access-client-schema.md).

## Goal

Every logged operator decision stores the request IP and user agent. JSON, CSV, and the GUI table show them.

## Changes

- `maybeLogOperatorAccess` copies `c.ClientIP()` and `c.GetHeader("User-Agent")` onto the record. Truncate UA at 512 bytes at this boundary.
- Append `ip_address` and `user_agent` to `accessLogCSVHeader` and `accessLogCSVRow`.
- Add columns on `web/templates/partials/operator_access_logs.tmpl`.
- httptest that a deny from a JSON caller persists the client IP.

Do not log ordinary read allows. Insert policy is unchanged.

## Data structures

No new types. Middleware fills the two strings. Repository trusts them.

## Verification

Static: `go test -count=1 ./internal/middleware ./internal/admin ./internal/coreapp ./web`.

Runtime: httptest viewer POST `/admin/tenants` with `X-Forwarded-For` and a User-Agent. Access log row has that IP and UA, decision deny.
