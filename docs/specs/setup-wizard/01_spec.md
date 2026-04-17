# `jotter setup` wizard — v1 spec

## Why

v0.1.0 shipped prebuilt binaries and a `curl | sh` install script, so a user can now get `jotter` on their PATH in one command. But the distance from "binary installed" to "`/start` works in a Claude Code session" is still six manual steps:

1. Create a git-backed data repo somewhere
2. Optionally add a remote and push
3. Write a `.jotter` config pointing at it
4. Install five skill files into `~/.claude/skills/`
5. Grant the `Bash(jotter:*)` permission in `~/.claude/settings.json`
6. Run a write + tail by hand to confirm everything's wired

Each step is a place where a user gives up. `jotter setup` collapses all six into one prompted flow.

## In scope (v1, target v0.2.0)

- `jotter setup` — an interactive Go subcommand that takes a user from "binary on disk" to "`/start` works" in one flow
- **Claude Code only** — other agents (Codex, Aider, Cursor) are out of scope for v1; structure the code so they slot in later without rewrites
- Idempotent — re-running detects existing state and updates rather than clobbering
- Ships in v0.2.0 (the binary — not a separate release artefact)

## Out of scope

- Multi-agent support — parked until v0.2.0 ships and there's a working single-agent baseline
- Auto-upgrade (`jotter setup --upgrade`)
- Uninstall (`jotter setup --uninstall`)
- Remote-repo creation via `gh repo create` — user brings their own remote URL

## Target user flow

```bash
# user already has jotter on PATH (via install.sh or go install)
jotter setup
```

That's it. The wizard detects current state and prompts only when a decision is needed.

## Wizard steps

Each step detects current state first, skips if already satisfied, prompts only when there's a decision. Any step failure short-circuits — re-running picks up where it broke.

1. **Claude Code present?** — check for `~/.claude/`. If missing, print the Claude Code install docs link and exit cleanly (not a failure, just out of scope).
2. **Data directory** — prompt for path (default: `~/session-logs-data`). Existing git repo → adopt. Existing non-git dir → offer `git init`. Missing → create + init.
3. **Git remote** — prompt for optional remote URL. Skipping is fine (finish entries won't push). If provided, wire it up and do a first push to validate.
4. **`.jotter` config** — write `~/.jotter` pointing at the chosen data dir. Existing `.jotter` → prompt before overwriting. Detect symlinks (dotfiles setups) and warn before writing through them.
5. **Skills install** — copy the five embedded skills into `~/.claude/skills/`. Detect existing skills and prompt before overwriting. Detect symlinks and warn.
6. **Claude Code permission** — merge `Bash(jotter:*)` into `~/.claude/settings.json`'s `permissions.allow` list. Never clobber existing settings.
7. **Smoke test** — run `jotter write --type note --content "setup smoke test"` then `jotter tail --limit 1` and show output. Success → print "try `/start` in a Claude session" hint.

Each step reports ✓ / skipped / updated / failed. Final summary table at the end.

## Decisions

These were open questions during scoping and have been closed out:

- **Interactivity library: `charmbracelet/huh`.** Actively maintained (survey/v2 was archived 2024-04-07), form abstraction maps cleanly to the step model, same ecosystem as bubbletea if we want a richer TUI later.
- **Skills shipping: `//go:embed` from the top-level `main` package.** Binary is self-contained; skills version in lockstep with the binary; no install-path discovery. Go's embed rules forbid `..`, so the embed directive must be at or above the `skills/` directory — embedding from `main.go` keeps skills discoverable at the repo root.
- **No Windows for v1.** GoReleaser matrix stays darwin/linux × amd64/arm64 (decided with release-infra).

## Acceptance criteria

1. On a machine with Claude Code and `jotter` installed but no data repo and no skills:
   - `jotter setup` completes in fewer than five prompts
   - Afterwards, `/start` in a Claude Code session works on the first try with no manual config
2. Running `jotter setup` a second time is idempotent — detects existing state, updates only what needs updating, never destroys user data.
3. On a machine without Claude Code, `jotter setup` exits cleanly with a pointer to the Claude Code install docs (exit 0, not a failure).
4. The full path — `install.sh` → `jotter setup` → first `/start` — completes in under five minutes on a fast connection.

## Next stage

- `02_plan.md` — code layout, `huh`-based step framework shape, embed wiring, testing strategy, order of work
