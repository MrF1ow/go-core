# Operator IAM deferred

Parked after the 2026-08-30 key lifecycle plan. Do not start these in that program. Move an item into a new plan when its unlock condition is true. Do not treat this file as a backlog to burn down.

## Per-app admin API keys

An admin key with a non-null `app_id` that becomes `Principal.AppID` and hits `/admin`.

Blocked by

- JSON `/admin` handlers do not filter by `Principal.AppID`. GUI does, in `internal/admin/gui_scope.go`. Copying AppID onto a JSON principal before that work is a cross-app read.
- `api_keys.app_id` is `ON DELETE CASCADE`. Bound admin keys would die with the application. GUI accounts use `RESTRICT`.
- Accounts cannot be superadmin and bound. Keys would be, unless a new CHECK mirrors `admin_accounts_superadmin_is_platform`.

Unlock when JSON has the same restrict as GUI, operator binding uses `RESTRICT` (not the worker FK), and superadmin keys stay platform.

Keep `api_keys_admin_app_id_null` until then.

## App-type keys as operator principals

`X-App-API-Key` workers that `Has()` and ride `requireOp`.

Blocked by a second auth world. App keys use scopes on `/app/:id`. Email on those routes is a platform resource. Wrapping them in `Allows` would 403 send-email. Do not mix worker scopes into `NewPrincipal`.

Unlock when product wants workers to be operators, with a written answer for email, IP rules, and `KindAPIKey`.

## Many-to-many operator-to-app grants

One account or key scoped to a set of apps.

Blocked by no two-app hire. Username and email are globally unique. Disable is one-way. One human on two apps without platform cannot share `admin@`. A join table is that product, not a more future-proof catalog. `Has` would stay one role. Last-superadmin would stay a count of platform principals.

Unlock when a real operator must use two apps without a platform account.

## JSON mint of app-type keys

`POST /admin/operator/keys` in the lifecycle program mints `key_type=admin` only. App keys stay GUI `POST /gui/api-keys`.

Unlock when automation needs worker keys and the JSON body can force `app_id` without minting an admin key.

## Duration preset buttons

30, 90, or 365 click targets on admin key create. Create already defaults to now plus 90 days.

Unlock when operators miss the default enough to measure.

## JSON list filters for bound principals

A `gui_scope` twin on `/admin`. Not needed while admin keys stay platform.

Unlock as a prerequisite of per-app admin keys, in the same program as that CHECK drop, never as a later patch.

## Last-superadmin count then update

Demote and disable count, then write, with no transaction. Two concurrent disables can clear the last platform superadmin.

Unlock when that race is observed or when the next IAM mutation PR touches those handlers anyway.

## GUI operator account hard delete

`AccountRepository.DeleteByID` has no HTTP caller. Evidence FKs are `ON DELETE SET NULL`. Passkeys CASCADE.

Do not add a delete button. Disable is the leaver path.

## `Grants()` lattice, Redis grant store, cannot-grant-above-self

Rejected in the shipped IAM recap. Still rejected.

## SOC 2 Type I/II organizational evidence

Not a go-core feature. Out of this repository.
