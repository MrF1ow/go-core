# Phase 8: GUI mutations

[Overview](overview.md)

PR: GUI mutations. Base is GUI evidence pages and CSRF HTML.

## Goal

A superadmin cookie can create a viewer operator, change a role, and disable an account from the roster page. Last superadmin still 409s. Tampered POST without `admin_iam:write` is HTML 403.

## Changes

- GUI forms that call the same domain paths JSON already has: create account (always viewer), PUT account role, POST disable, PUT key role. Copy the API-key role stamp in `gui_handler.go`, not tenant CRUD.
- `requireGUI(admin_iam, write)` on those registrations. Form GETs that start a mutation are `:write`. List and export stay `:read`.
- Write CTA omit via `Can`. Admin has no `admin_iam:read`, so that cookie never reaches the page. Superadmin sees Create and Disable. Do not show those buttons without write.
- Last-superadmin is HTML 409, not JSON. JSON `OperatorAccountRole` and `OperatorDisableAccount` stay JSON 409. Reuse `WouldLeaveLastSuperadmin`.
- Actor on events is the GUI principal. Reuse `writeIAMEvent`.
- CSRF HTML from phase 5 covers a missing token. Do not add a second CSRF check.

Do not add delete. Disable is the leaver path. Do not add custom-role create here.

## Data structures

No new domain types. HTML 409 body uses the same last-superadmin message as JSON `dto.ErrorResponse`, rendered as an alert fragment.

## Verification

Static: `go test -count=1 ./internal/admin ./internal/coreapp ./internal/middleware`.

Runtime: superadmin cookie POST create → 201/200, role viewer, event `create_principal`. Last superadmin disable → 409 HTML, still enabled. Viewer crafted POST → 403, no event. CSRF-missing POST → 403 HTML, no `X-GUI-Forbidden`.
