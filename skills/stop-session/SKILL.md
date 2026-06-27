---
name: stop-session
description: Stop a work session — commit any dirty work, write a stop entry to the jotter log. Leaves a walk-away state. Use when the user says "/stop", "let's stop for today", "let's stop for the morning", "let's wrap up", "wrap up", "end this session", "let's call it", "that's enough for today", or the legacy "/finish".
---

# Stop Session

End-of-session wrap. **Walk-away guarantee:** when this skill completes, dirty work is committed and the stop entry is written to the jotter log. The skill terminates after the write — no surprise follow-up actions.

A `stop` marks the end of a *work session*, not the end of the branch — you'll likely stop a branch many times. When a branch is finished for good, use `handover` instead. (`stop` was previously called `finish`; that type still works and `/finish` still triggers this skill.)

**Mid-session break, not the end?** Use `break-session` instead.

Layer your own end-of-session conventions (ROADMAP updates, doc refresh, PR housekeeping, cron cancellation) on top of this skill — do them **before** invoking `/stop` so the stop entry can accurately reference what was done.

---

## Steps

### 1 — Determine project and branch

```bash
PROJECT=$(jotter project)
BRANCH=$(jotter branch)
```

### 2 — Commit dirty work

```bash
git status
git diff --stat
```

If the tree is clean, skip to step 3. Otherwise propose a commit grouping, wait for the user to approve, and commit. One commit per logical change.

### 3 — Preview the stop entry

Render the draft back to the user as a quoted block **before** writing, so they can see what's about to land in the log. Summarise the session — what was built or fixed, key decisions, gotchas. The `--next` field is the handover: 2-3 priorities, in order.

> **Content:**
> - <bullet 1: what shipped>
> - <bullet 2: key decision>
> - <bullet 3: gotcha or debt>
>
> **Next:**
> - <priority 1>
> - <priority 2>

### 4 — Write — final action of the skill

```bash
jotter write \
  --project "$PROJECT" \
  --branch "$BRANCH" \
  --type stop \
  --content "<bullets from preview>" \
  --next "<priorities from preview>"
```

`jotter write --type stop` auto-commits the data repo locally. The push happens asynchronously — the background timer (`jotter daemon`) carries the entry to the remote within its interval, so the write never blocks on the network. To push immediately, run `jotter sync`.

### 5 — Confirm

> "Stopped at HH:MM. Tree clean, log written. Safe to close the laptop."

No further actions after this — the skill is done.
