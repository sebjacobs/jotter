---
name: setup-jotter
description: Set up jotter — configure the data repo and add Claude Code permissions. Use when setting up jotter for the first time after installing the plugin.
---

# Setup Jotter

Post-install setup for the jotter plugin. The plugin delivers the skills; this skill installs the `jotter` binary and handles the user-specific configuration (data repo, config file, Claude Code permissions).

Each step checks preconditions before acting — safe to re-run if setup was interrupted.

---

## Steps

### 1 — Install the `jotter` binary

Check whether `jotter` is already on `PATH`:

```bash
command -v jotter
```

If present, skip to step 2.

Otherwise, check for a Go toolchain:

```bash
command -v go
```

**If Go is missing,** stop and tell the user:

> "Jotter is built from source and needs Go 1.22+. Install it first:
> - macOS: `brew install go`
> - Linux: use your distro's package manager, or download from https://go.dev/dl/
> - Other: https://go.dev/dl/
>
> Then re-run `/setup-jotter`."

**If Go is present,** install jotter:

```bash
go install github.com/sebjacobs/jotter@latest
```

Verify the binary is callable:

```bash
command -v jotter
```

If the above prints nothing, the Go bin dir isn't on `PATH`. Tell the user which shell rc file to update:

```bash
echo "Add this to your shell rc: export PATH=\"$(go env GOPATH)/bin:\$PATH\""
```

Then have them open a new shell (or `source` the rc file) and re-check `command -v jotter` before moving on.

---

### 2 — Choose or create the data repo

Jotter stores session logs as JSONL files in a git repository. Ask the user:

> "Where should jotter store session logs? This needs to be a git repository — it can be an existing one or I can create a new one."
>
> Provide a path (e.g. `~/session-logs`):

Once they provide a path:

**If the path exists and is a git repo** — confirm and move on.

**If the path exists but isn't a git repo:**

```bash
cd <path> && git init
```

**If the path doesn't exist** — create it and initialise:

```bash
mkdir -p <path> && cd <path> && git init
```

Then ask if they want a private GitHub remote:

> "Want me to create a private GitHub repo for backup? (recommended)"

If yes:

```bash
cd <path>
gh repo create <repo-name> --private --source=. --push
```

---

### 3 — Write the config file

Write the data directory path to `~/.config/jotter/config`:

```bash
mkdir -p ~/.config/jotter
echo "<data-dir-path>" > ~/.config/jotter/config
```

Verify it works:

```bash
jotter ls
```

This should run without error (may show "no projects found" if the repo is empty, which is fine).

---

### 4 — Add Claude Code permission

Jotter needs to run as a Bash command without requiring approval each time. Add `Bash(jotter *)` to the user's Claude Code settings.

Ensure the settings file exists (create with an empty JSON object if missing):

```bash
mkdir -p ~/.claude
[ -f ~/.claude/settings.json ] || echo '{}' > ~/.claude/settings.json
```

Then merge the permission in idempotently:

```bash
python3 -c "
import json, pathlib
path = pathlib.Path.home() / '.claude' / 'settings.json'
settings = json.loads(path.read_text() or '{}')
perms = settings.setdefault('permissions', {}).setdefault('allow', [])
if 'Bash(jotter *)' in perms:
    print('Permission already present')
else:
    perms.append('Bash(jotter *)')
    path.write_text(json.dumps(settings, indent=2) + '\n')
    print('Added Bash(jotter *) permission')
"
```

If `python3` isn't available, ask the user to add `\"Bash(jotter *)\"` to the `permissions.allow` array in `~/.claude/settings.json` manually.

---

### 5 — Smoke test

Run a quick end-to-end test to verify everything works:

```bash
jotter write --project _setup-test --branch main --type start --content "Setup verification"
jotter tail --project _setup-test --branch main --limit 1
```

If both commands succeed, the setup is complete. The test entry stays in the `_setup-test` project — harmless, and `jotter ls` will still group it separately from real work.

---

## Done

> "Jotter is set up and working. Here's what to do next:"
>
> - Run `/start` at the beginning of each coding session
> - Run `/finish` when you're done
> - Use `/save` for mid-session checkpoints
> - Use `/break` before stepping away
>
> Session logs are stored in `<data-dir-path>` and auto-committed to git.
