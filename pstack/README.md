# pstack (go-core snapshot)

Trimmed Cursor pstack plugin for this repo. Live slash skills load from the plugin (`pstack/.cursor-plugin/plugin.json`) and from `.cursor/settings.json`. The model map is `.cursor/rules/pstack-models.mdc`. Cloud Agent `install`/`start` copy that file to `~/.cursor/rules/pstack-models.mdc` so skills that Read the home path still hit this map.

Do not run `/setup-pstack` against upstream defaults. Edit the project rule, then reboot or copy it to the home path.

Upstream: pstack 0.14.8. Autopilot playbooks, Benny Slack automations, TypeScript auto-skill, make-bot-ui, the generic guide, and unused why-source adapters are not in this snapshot.

If the Cursor marketplace pstack plugin is also enabled on the account, Cloud Agents still inject that copy. Disable the marketplace plugin if you want only this snapshot. The model map still wins either way once start has copied it to `~/.cursor/rules/pstack-models.mdc`.
