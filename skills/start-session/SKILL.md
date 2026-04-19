---
name: start-session
description: Restore context from the jotter log and write a start entry. Use when the user says "/start", "/start-session", "let's start", "start session", "begin session", or at the start of any longer session.
---

# Start Session

Restores context from prior jotter entries and records a start entry for the new session. Layer any session-management conventions (time-boxing, pacing, goal-setting) on top of this skill — it only handles the logging side.

---

## Steps

### 1 — Determine project and branch

```bash
basename "$(git rev-parse --show-toplevel)"
git rev-parse --abbrev-ref HEAD
```

### 2 — Check whether the previous session ended cleanly

```bash
jotter tail --project <project> --branch <branch> --limit 1
```

If the last entry is **not** a `finish` (i.e. it's a `start`, `checkpoint`, or `break`), the previous session likely crashed or skipped `/finish`. Offer to run `/recover` before continuing.

### 3 — Restore context from the log

First check whether the branch has a log — avoids an error from `tail` when there's nothing to read:

```bash
jotter ls --project <project>
```

If the current branch has entries, read the last few:

```bash
jotter tail --project <project> --branch <branch> --limit 5
```

If the current branch isn't listed but `main` is, fall back for broader project context:

```bash
jotter tail --project <project> --branch main --limit 3
```

If neither exists, skip to step 4 — no prior context to restore.

Surface the most recent finish entry's `**Next:**` field — that's the handover prompt from the last session.

### 4 — Write the start entry

```bash
jotter write \
  --project <project> \
  --branch <branch> \
  --type start \
  --content "<what this session is picking up or starting>"
```
