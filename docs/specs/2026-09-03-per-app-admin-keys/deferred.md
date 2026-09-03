# Per-app admin keys deferred

Parked after the 2026-09-03 per-app admin keys plan. Do not start these in that program. Move an item into a new plan when its unlock condition is true. Do not treat this file as a backlog to burn down.

## App-type keys as operator principals

`X-App-API-Key` workers that `Has()` and ride `requireOp`.

Blocked by a second auth world. App keys use scopes on `/app/:id`. Email on those routes is a platform resource. Wrapping them in `Allows` would 403 send-email. Do not mix worker scopes into `NewPrincipal`.

Unlock when product wants workers to be operators, with a written answer for email, IP rules, and `KindAPIKey`.

## Many-to-many operator-to-app grants

One account or key scoped to a set of apps.

Blocked by no two-app hire. Username and email are globally unique. Disable is one-way. One human on two apps without platform cannot share `admin@`. A join table is that product, not a more future-proof catalog. `Has` would stay one role. Last-superadmin would stay a count of platform principals.

Unlock when a real operator must use two apps without a platform account.

## JSON mint of app-type keys

`POST /admin/operator/keys` mints `key_type=admin` only. App keys stay GUI `POST /gui/api-keys`.

Unlock when automation needs worker keys and the JSON body can force `app_id` without minting an admin key.

## Duration preset buttons

30, 90, or 365 click targets on admin key create. Create already defaults to now plus 90 days.

Unlock when operators miss the default enough to measure.

## Last-superadmin count then update

Demote and disable count, then write, with no transaction. Two concurrent disables can clear the last platform superadmin.

Unlock when that race is observed or when the next IAM mutation PR touches those handlers anyway. bind does not touch disable or demote. Do not hitchhike.

## GUI operator account hard delete

`AccountRepository.DeleteByID` has no HTTP caller. Evidence FKs are `ON DELETE SET NULL`. Passkeys CASCADE.

Do not add a delete button. Disable is the leaver path.

## GUI mint of bound admin keys

Platform GUI create still stores null `app_id`. Bound GUI operators still cannot create admin keys.

Unlock when humans, not automation, must mint a bound admin key from the API Keys page.

## `Grants()` lattice, Redis grant store, cannot-grant-above-self

Rejected in the shipped IAM recap. Still rejected.

## SOC 2 Type I/II organizational evidence

Not a go-core feature. Out of this repository.
