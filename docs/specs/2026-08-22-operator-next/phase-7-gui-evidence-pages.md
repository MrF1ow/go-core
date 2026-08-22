# Phase 7: GUI evidence pages

[Overview](overview.md)

PR: GUI evidence pages. Base is GUI roster. Shared `gui_handler` and nav. Read-only.

## Goal

The Operator IAM page can show IAM events and access logs, not only the roster. JSON list types are the only rows.

## Changes

- HTMX tabs or sibling paths under `/gui/operator`: roster (phase 6), events, access logs. One nav row still. Do not add three sidebar links.
- `GET /gui/operator/iam-events` and `GET /gui/operator/access-logs` with `requireGUI(admin_iam, read)`.
- Filters JSON already has: events by target key or account id, access logs by decision.
- Reuse `ListIAMEvents` and `ListAccessLogs`. Do not add GUI SQL.
- Export buttons reuse phase 3 JSON CSV routes or GUI wrappers of them.

Do not add mutations. Do not log these GETs as IAM history.

## Data structures

Reuse `IAMEvent` and `AccessRecord`. One page constructor from phase 6. Tab id is the only extra field.

## Verification

Static: `go test -count=1 ./internal/admin ./internal/coreapp ./web`.

Runtime: superadmin cookie GET events → 200, newest first. Viewer → 403 HTML with the header. Decision=deny filter on access logs returns only denies. Inventory sees both new `requireGUI` lines.
