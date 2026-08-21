# Phase 6: Sidebar and write CTAs

[Overview](overview.md)

## Goal

A viewer does not see doors they cannot open. First paint is the only sidebar paint. Route `Has` still applies on the next request.

## Changes

- `base.tmpl` ranges `NavGroups`. Drop the hardcoded headings and links. Keep the logo, footer My Account, theme toggle, and logout outside the range. Keep the existing HTMX swap attributes on each item (`hx-target="#page-content"`, `hx-select="#page-content"`, `hx-push-url`).
- Title map in the sidebar script can stay as a static JS object. It does not grant access.
- Omit write CTAs via `Can` on the templates this slice already has to touch to make the merge bar true. At minimum:
  - Create / New buttons on list pages whose resource the viewer lacks (`tenants.tmpl` and the other `hx-get=.../new` list pages).
  - Users Import button and import modal entry.
  - User detail sessions widget. Viewer `users:read` without `sessions:read` should not fire `hx-get` on `#user-sessions-container`. Support still loads it.
  - User detail write actions (toggle, unlock, unlink, passkey delete, revoke devices) behind `users:write`.
  - API key create button for `api_keys:write`. Operator role `<select>` only when `admin_iam:write`. Tampered POST still 403 from phase 4.
  - List-row Edit / Delete / Revoke links on fragments that a read-only role can still load (tenant list, user list, and so on). If a list fragment has no `TemplateData.Can`, pass a small `Can` closure or a bool set from the handler. Do not CSS-hide.
- Do not re-render the sidebar on HTMX fragment swaps. `hx-select="#page-content"` already leaves the nav alone.

Land this in the same merge window as phase 5. A wired GUI with every Create Tenant button still visible is not mergeable.

## Data structures

No new types. Templates call `Can` with the same resource/action strings as `NavSpec` and the route map.

## Verification

Static: `go test -count=1 ./internal/admin ./web`.

Runtime, body assertions on httptest HTML:

- Viewer GET `/gui/` (or `/gui/users` after phase 2) sidebar HTML contains Users and does not contain Tenants or `sidebar-heading">Email`.
- Viewer GET `/gui/users` does not contain the Import button.
- Viewer GET `/gui/users/:id` does not contain `user-sessions-container` with an `hx-get` to sessions. Support GET does.
- Admin GET `/gui/api-keys/new` does not contain `name="operator_role_id"` as a visible select of superadmin. Superadmin GET does. (Hidden current-role field on edit is allowed.)
- Superadmin sidebar still has Tenants, Settings, and API Keys.
