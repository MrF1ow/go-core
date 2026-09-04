# Per-app admin keys deferred

Parked after the 2026-09-03 per-app admin keys plan. json-scope and bind shipped. Do not treat this file as a backlog to burn down. Move an item into a new plan when its unlock condition is true. Never items have no unlock.

## Last-superadmin count then update

Demote and disable count, then write, with no transaction. Two concurrent disables can clear the last platform superadmin. Disable is one-way. Waiting until the race is observed waits until those accounts are already gone.

json-scope and bind did not touch disable or demote. Do not hitchhike onto an unrelated mint PR.

Unlock when the next PR mutates disable or demote, or as a dedicated `BEFORE UPDATE` trigger in the `023` style. The trigger raises when an update would leave zero enabled platform superadmins. Do not wrap four handlers in transactions. Count stays platform GUI accounts with `app_id IS NULL`.

## Many-to-many operator-to-app grants

One account or key scoped to a set of apps.

Parked for missing product demand, not because username and email are unique. Unique `admin@` is why one account would need two apps. A join table is that product. `Has` would stay one role. Last-superadmin would stay a count of platform principals.

Unlock when a real operator must use two apps without a platform account.

## GUI mint of bound admin keys

Platform GUI create still stores null `app_id`. Bound GUI operators still cannot create admin keys. JSON `POST /admin/operator/keys` with `app_id` is the bind path. Do not add a picker because bind shipped.

Unlock when a named GUI operator cannot use JSON and must mint a bound admin key from the API Keys page.

## App-type keys as operator principals

Never. Bound admin keys are the machine principal for app-scoped operator access. `X-App-API-Key` workers authenticate on `/app/:id` and never build a `Principal`. `Allows` would 403 `ResEmail` while `/app/:id/send-email` is a live worker route. `KindAPIKey` already means admin keys. Do not mix worker scopes into `NewPrincipal`.

## JSON mint of app-type keys

Never on `POST /admin/operator/keys`. Bind owns `app_id` on that body. Empty is a platform admin key. Set is a bound admin key. Posted `key_type=app` still inserts `key_type=admin`. Worker JSON mint, if it ever exists, is a new route. App keys stay GUI `POST /gui/api-keys`.

## Duration preset buttons

Never. Create already fills now plus 90 days and requires admin expiry. Missing the default is not evidence for 30, 90, or 365 click targets.

## GUI operator account hard delete

Never. Disable is the leaver path. `AccountRepository.DeleteByID` has no HTTP caller. Evidence FKs are `ON DELETE SET NULL`. Passkeys CASCADE. Do not add a delete button. Delete the unused `DeleteByID` query on the next `admin_account.sql` PR if you are already there.

## `Grants()` lattice, Redis grant store, cannot-grant-above-self

Rejected in the shipped IAM recap. Still rejected.

## SOC 2 Type I/II organizational evidence

Not a go-core feature. Out of this repository.
