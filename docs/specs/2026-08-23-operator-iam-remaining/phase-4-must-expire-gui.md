# Phase 4: Must-expire GUI

[Overview](overview.md)

PR: Must-expire. Same PR as [phase 3](phase-3-must-expire-schema.md).

## Goal

The API Keys form cannot mint or clear a forever admin key. Create is a future datetime. Edit cannot persist null.

## Changes

- Replace the forever branch in `parseOptionalExpiresAtKeeping` for admin keys. Empty create is 400. Empty edit keeps `current` when `current` is non-null. Posting the same `datetime-local` string still keeps a past instant so name and role edits on expired keys work.
- Create form `datetime-local` defaults to now plus 90 days. Helper text no longer says leave blank for forever.
- App key create and edit still allow empty expiry.
- httptest admin create with empty `expires_at` is 400 and no row. Admin create with the default field succeeds and roster shows a non-null expiry. Edit that clears the field keeps the stored instant.

Do not add JSON key create. Do not add 30/90/365 preset buttons in this phase.

## Data structures

Keep `*time.Time` on the key model. Admin persist paths never pass nil after parse.

## Verification

Static: `go test -count=1 ./internal/admin ./internal/coreapp`.

Runtime: superadmin cookie POST `/gui/api-keys` as admin type with empty expiry. 400, no row. POST with a future datetime. 200, roster `expires_at` set. App-key create with empty expiry still succeeds.
