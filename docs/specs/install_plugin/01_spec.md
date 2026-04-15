# Install Plugin — Spec

## What

A Claude Code skill (`/setup-jotter`) that guides a new user through installing and configuring jotter. Ships in the jotter repo so users can copy it into their `~/.claude/skills/` or run it directly.

## Why

Jotter requires several pieces to work together — the binary, a git-backed data repo, a config file, Claude Code permissions, and session skills. Setting this up manually is fiddly and error-prone. A guided skill makes adoption frictionless.

## Behaviour

The skill walks through these steps interactively:

### 1 — Install the binary

Check if `jotter` is already on PATH. If not, offer two options:
- `go install github.com/sebjacobs/jotter@latest` (requires Go)
- Download a pre-built binary from the latest GitHub release (future — requires CI release workflow)

Verify the install worked: `jotter --help`

### 2 — Choose or create the data repo

Ask the user: "Where should jotter store session logs? This should be a git repository."

Options:
- **Existing repo** — user provides a path, skill verifies it's a git repo
- **New repo** — user provides a path, skill runs `git init`, optionally `gh repo create` for a private remote

### 3 — Write the config file

Write the chosen path to `~/.config/jotter/config` (creating directories as needed).

Verify: `jotter ls` should run without error.

### 4 — Add Claude Code permission

Add `Bash(jotter *)` to the user's `~/.claude/settings.json` so jotter commands don't require approval each time.

### 5 — Install session skills

Copy the session management skills into `~/.claude/skills/`:
- `start-session`
- `finish-session`
- `save-session`
- `break-session`
- `recover-session`

These skills are the ones from the user's dotfiles that call jotter. The install plugin should either:
- Copy them from a `skills/` directory shipped in the jotter repo, or
- Provide a curl/clone command to fetch them

### 6 — Verify

Run a smoke test: `jotter write --project test --branch test --type start --content "setup complete"` then `jotter tail --project test --branch test --limit 1` to confirm the full pipeline works. Clean up the test entry.

## Out of scope

- CI release workflow (separate feature)
- Updating existing installations
- Windows support (darwin/linux only for now)

## Acceptance criteria

- [ ] Skill file exists at `install/SKILL.md` in the jotter repo
- [ ] Each step checks preconditions before acting (idempotent — safe to re-run)
- [ ] User is prompted before any write operation (no silent side effects)
- [ ] Works on macOS and Linux
