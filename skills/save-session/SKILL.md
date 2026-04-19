---
name: save-session
description: Write a mid-session checkpoint entry to the jotter log. Use when the user says "/save", "checkpoint", "save progress", or before risky operations like schema migrations, large refactors, or long-running tasks.
---

# Save Session

Writes a `checkpoint` entry to the jotter log — a snapshot of current progress and decisions without ending the session. Use before risky operations, or to preserve state before a `/clear`.

---

## Steps

### 1 — Determine project and branch

```bash
PROJECT=$(jotter project)
BRANCH=$(jotter branch)
```

### 2 — Read recent context (avoid duplication)

```bash
jotter tail --project "$PROJECT" --branch "$BRANCH" --limit 3
```

Review what's already been captured so the checkpoint adds new information rather than repeating earlier entries.

### 3 — Write the checkpoint

```bash
jotter write \
  --project "$PROJECT" \
  --branch "$BRANCH" \
  --type checkpoint \
  --content "<progress since last entry, decisions made, current state>" \
  --next "<what you're about to do next>"
```

Keep it concise — a snapshot, not a summary.
