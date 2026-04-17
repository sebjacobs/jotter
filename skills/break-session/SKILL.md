---
name: break-session
description: Save mid-session state before a break. Use when the user says "/break", "taking a break", "let's take a break", "back in a bit", "stepping away", "pausing", or similar.
---

# Break Session

Quick checkpoint before stepping away. Captures current state, commits any dirty work, cancels the session timer.

---

## Steps

### 0 — Get context

Run `date` to get the current time.

Determine the project name and branch:

```bash
basename "$(git rev-parse --show-toplevel)"
git rev-parse --abbrev-ref HEAD
```

### 1 — Write the break entry

Summarise current progress in a few bullet points, then write it:

```bash
jotter write \
  --project <project> \
  --branch <branch> \
  --type break \
  --content "<what's been done, current state, anything half-finished>" \
  --next "<what to pick up on return>"
```

### 2 — Commit dirty work

```bash
git status
git diff --stat
```

If there are uncommitted changes worth saving, propose a grouping to the user and commit after approval. Don't leave half-done work uncommitted across a break.

### 3 — Cancel session timer

If a session cron timer is running, cancel it with `CronDelete <job-id>`. `/start` will set a fresh one on return.

### 4 — Confirm

> "Break saved. Run `/start` when you're back."
