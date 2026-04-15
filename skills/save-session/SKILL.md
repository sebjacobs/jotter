---
name: save-session
description: Mid-session checkpoint — snapshot current decisions and progress without archiving or cleaning up. Use when the user says "/save", "checkpoint", "save progress", or before risky operations like schema migrations, large refactors, or long-running tasks.
---

# Save Session

Mid-session checkpoint. Captures current progress and decisions without the full end-of-session routine. Does **not** archive, move roadmap items, or propose commits.

Use before risky operations (migrations, large refactors) or when you want to preserve state before a `/clear`.

---

## Steps

### 0 — Get context

Run `date` to get the current time.

Determine the project name and branch:

```bash
basename "$(git rev-parse --show-toplevel)"
git rev-parse --abbrev-ref HEAD
```

### 1 — Read recent context

```bash
jotter tail --project <project> --branch <branch> --limit 3
```

Review the last few entries to understand what's already been captured — avoid duplicating.

### 2 — Write the checkpoint

```bash
jotter write \
  --project <project> \
  --branch <branch> \
  --type checkpoint \
  --content "<progress since last entry, decisions made, current state>" \
  --next "<what you're about to do next>"
```

Keep it concise — a few bullet points per topic. This is a snapshot, not a session summary.

### 3 — Confirm

> "Checkpoint saved at HH:MM. Safe to `/clear` or continue."

Do **not** propose commits, update the roadmap, or archive anything. That's `/finish`'s job.
