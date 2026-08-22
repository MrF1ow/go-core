# Phase 4: Evidence retention

[Overview](overview.md)

PR: Evidence retention. Base is Evidence export. No new columns. No sqlc.

## Goal

Operator evidence older than 365 days is deleted on a schedule. The request path does not wait on that delete.

## Changes

- Cleanup sibling to `internal/log/cleanup.go`, or a second delete loop inside that worker. Operator tables have one window, so `DELETE WHERE at < $1` in batches, not `expires_at`.
- Default 365 days for both `operator_access_logs` and `operator_iam_events`. Same number as critical activity-log retention. Document it in [activity-logging.md](../../activity-logging.md): operator evidence, v1 insert policy, 365-day cleanup, CSV export paths.
- Failed delete logs and the next tick retries. Idempotent: a second run deletes whatever is still older than the cutoff.
- Do not add settings keys in v1 unless activity-log cleanup already requires a config struct the sibling can share. Hard-code 365 if a new key would be the only consumer.
- Do not UPDATE rows. Append-only stays append-only. Cleanup is delete-by-age, not expiry-stamp.

Do not clean `activity_logs` in this PR except by calling the existing worker.

## Data structures

Cutoff is `time.Now().UTC().AddDate(0, 0, -365)`. Batch size matches activity-log cleanup.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/log ./internal/coreapp`.

Runtime: insert a row with `at` 366 days ago (or inject the cutoff), run cleanup once, row gone. A row from now remains. Second run deletes zero of the remaining new row.
