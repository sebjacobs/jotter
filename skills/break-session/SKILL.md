---
name: break-session
description: Save mid-session state before stepping away — auto-WIP-commit any dirty work, write a break entry to the jotter log. Walk-away guarantee in under 30 seconds. Use when the user says "/break", "taking a break", "let's take a break", "back in a bit", "stepping away", "pausing", or similar.
---

# Break Session

Quick wrap before stepping away. **Walk-away guarantee:** when this skill completes, dirty work is committed (as a single WIP commit, no ceremony) and the break entry is written. Optimised for ASAP — the user is walking away now.

---

## Steps

### 1 — Determine project and branch

```bash
PROJECT=$(jotter project)
BRANCH=$(jotter branch)
```

### 2 — WIP-commit dirty work

```bash
git status --short
```

If the tree is clean, skip to step 3.

Otherwise stage everything and commit as a single WIP commit. **Do not propose groupings or wait for approval** — the user is stepping away now, ceremony defeats the purpose. The commit can be reworked on return.

```bash
git add -A
git commit -m "WIP: break at $(date +%H:%M)"
```

### 3 — Preview the break entry

Render the draft back to the user as a quoted block before writing:

> **Content:**
> - <what's been done, current state, anything half-finished>
>
> **Next:** <what to pick up on return>

### 4 — Write — final action of the skill

```bash
jotter write \
  --project "$PROJECT" \
  --branch "$BRANCH" \
  --type break \
  --content "<content from preview>" \
  --next "<next from preview>"
```

The `--next` field is what `/start` will surface when the session resumes.

### 5 — Confirm

> "Break saved at HH:MM. WIP committed, log written. Run `/start` when you're back."

No further actions after this.
