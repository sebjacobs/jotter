---
name: setup-jotter
description: Set up jotter — configure the data repo and add Claude Code permissions. Use when setting up jotter for the first time after installing the plugin.
---

# Setup Jotter

Post-install setup for the jotter plugin. The plugin delivers the binary and skills automatically — this skill handles the user-specific configuration that can't be automated.

Each step checks preconditions before acting — safe to re-run if setup was interrupted.

---

## Steps

### 1 — Choose or create the data repo

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

### 2 — Write the config file

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

### 3 — Add Claude Code permission

Jotter needs to run as a Bash command without requiring approval each time. Add the permission to the user's Claude Code settings:

```bash
cat ~/.claude/settings.json
```

Check if `Bash(jotter *)` is already in the `permissions.allow` list. If not, add it:

```bash
# Read current settings, add permission, write back
python3 -c "
import json
path = '$HOME/.claude/settings.json'
with open(path) as f:
    settings = json.load(f)
perms = settings.setdefault('permissions', {}).setdefault('allow', [])
if 'Bash(jotter *)' not in perms:
    perms.append('Bash(jotter *)')
    with open(path, 'w') as f:
        json.dump(settings, f, indent=2)
    print('Added Bash(jotter *) permission')
else:
    print('Permission already exists')
"
```

---

### 4 — Smoke test

Run a quick end-to-end test to verify everything works:

```bash
jotter write --project _setup-test --branch main --type start --content "Setup verification"
jotter tail --project _setup-test --branch main --limit 1
```

If both commands succeed, the setup is complete. The test entry will remain in the logs — it's harmless.

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
