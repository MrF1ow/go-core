# Incident & Postmortem Context

Not a separate source, a **cross-cutting angle**. Incidents often motivate defensive code ("we added this check after the X outage"), so if the target looks defensive (null checks, retry logic, timeout handling, rate limiting, feature flags), hunt for incident history in git and `gh`:

- **Git**: commits with messages like "fix for incident", "add defensive check", "revert" followed by "re-apply with..." are strong signals
- **GitHub Issues / PRs**: `gh issue list`, `gh issue view`, and `closingIssuesReferences` on the PR that shipped the target
- **In-repo docs**: `docs/`, `SECURITY.md`, `CHANGELOG.md`, and comments near the defensive check

If you find an incident link, fetch the full write-up. Postmortems typically have an "Action Items" section that ties directly to code changes.

Worth spending time on when the code's defensive character makes an incident-driven origin plausible. Skip it for code that doesn't look defensive.
