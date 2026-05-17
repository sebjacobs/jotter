---
name: save-session
description: Mid-session checkpoint — commit any dirty work, snapshot decisions and progress to the jotter log. Walk-away guarantee. Use when the user says "/save", "checkpoint", "save progress", "jot it down", "make a note", or before risky operations like schema migrations, large refactors, or long-running tasks.
---

# Save Session

Mid-session checkpoint. **Walk-away guarantee for the checkpoint mode:** when the skill completes, dirty work is committed and the log entry is written. Safe to `/clear`, walk away, or continue.

Use before risky operations (migrations, large refactors) or before `/clear`.

---

## Two modes

### Checkpoint (`/save`, "checkpoint", "save progress")

Progress so far plus what's next.

#### 1 — Determine project and branch

```bash
PROJECT=$(jotter project)
BRANCH=$(jotter branch)
```

#### 2 — Commit dirty work

```bash
git status
git diff --stat
```

If the tree is clean, skip to step 3. Otherwise propose a commit grouping, wait for the user to approve, and commit. These are proper commits (use `/break` for WIP).

#### 3 — Preview the checkpoint

Render the draft back to the user as a quoted block before writing:

> **Content:**
> - <progress since last entry, decisions made, current state — bullets>
>
> **Next:** <what you're about to do next>

Keep it concise — a snapshot, not a summary.

#### 4 — Write — final action of the skill

```bash
jotter write \
  --project "$PROJECT" \
  --branch "$BRANCH" \
  --type checkpoint \
  --content "<bullets from preview>" \
  --next "<next from preview>"
```

#### 5 — Confirm

> "Checkpoint saved at HH:MM. Tree clean, log written. Safe to /clear or continue."

### Note ("jot it down", "make a note", "note that")

Single observation, no handover, no commit. A casual jot to remember something — not a state snapshot.

#### 1 — Preview

> **Note:** <the thing to remember>

#### 2 — Write — final action

```bash
jotter write \
  --project "$(jotter project)" --branch "$(jotter branch)" \
  --type note \
  --content "<note from preview>"
```

#### 3 — Confirm

> "Noted at HH:MM."
