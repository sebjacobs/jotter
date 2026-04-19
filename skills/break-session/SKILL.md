---
name: break-session
description: Write a break entry to the jotter log before stepping away mid-session. Use when the user says "/break", "taking a break", "let's take a break", "back in a bit", "stepping away", "pausing", or similar.
---

# Break Session

Writes a `break` entry to the jotter log — a snapshot of current state and what to pick up on return. Use when stepping away mid-session without ending it.

---

## Steps

### 1 — Determine project and branch

```bash
basename "$(git rev-parse --show-toplevel)"
git rev-parse --abbrev-ref HEAD
```

### 2 — Write the break entry

```bash
jotter write \
  --project <project> \
  --branch <branch> \
  --type break \
  --content "<what's been done, current state, anything half-finished>" \
  --next "<what to pick up on return>"
```

The `--next` field is what `/start` will surface when the session resumes.
