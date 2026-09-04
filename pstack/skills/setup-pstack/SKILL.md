---
name: setup-pstack
description: Configure which models pstack uses per role. In this repo the map is already decided. Use for /setup-pstack, "configure pstack models", or changing pstack's model choices.
---

# Setup pstack

This repo's map lives at `.cursor/rules/pstack-models.mdc`. That file is the source of truth. Cloud Agent `install` and `start` copy it to `~/.cursor/rules/pstack-models.mdc` so skills that Read the home path still hit it.

Do not write upstream defaults (fable, opus 5, `-fast`, `-max`, `-xhigh`). Do not replace the project file with a template from this skill's old shape.

## Steps

### 1. Show the current map

Read `.cursor/rules/pstack-models.mdc`. Show every role and its model. That is the house map.

### 2. Change only if asked

If the user named specific roles to change, edit `.cursor/rules/pstack-models.mdc` only. Keep the constraints in that file's header. Real slugs must be ones this session can pass to `Task`. `inherit-parent` and `auto` always pass.

If they did not ask to change roles, leave the file alone.

### 3. Pin the home copy

Copy the project file to `~/.cursor/rules/pstack-models.mdc` so home-path reads match. Overwrite the home file. Never the other direction.

```bash
mkdir -p "$HOME/.cursor/rules"
cp .cursor/rules/pstack-models.mdc "$HOME/.cursor/rules/pstack-models.mdc"
```

### 4. Confirm

Tell the user the project file is the map, the home copy was refreshed, and new sessions pick it up. Re-running this skill without requested role changes only refreshes the home copy.
