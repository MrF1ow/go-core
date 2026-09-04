# Source playbooks

This snapshot keeps the sources this repo can actually query.

| Category | Playbook | Tools |
|---|---|---|
| Source control history | [`code-archaeology.md`](./sources/code-archaeology.md) | git, `gh` (PRs and GitHub Issues) |

Cross-cutting:

- [`incident-postmortem.md`](./sources/incident-postmortem.md). Add this if the target code looks defensive (null checks, retry, timeout, rate limit, feature flag, egress guard, OOM handler).

If an MCP for chat, tickets, observability, or analytics is actually enabled in the session, adapt the investigator prompt to that MCP. Do not look for Linear, Notion, Slack, Datadog, Sentry, or Databricks playbooks. Those files are not in this snapshot.
