---
name: handover
description: Distil a finished feature branch's log into a single handover entry on main, so its context survives the branch being deleted. Use when the user says "/handover", "this branch is done", "wrapping up this branch", "about to delete this branch", "hand this off", or just merged a branch and is about to remove it.
---

# Handover

A feature branch's jotter log is keyed by branch. When the branch is merged and deleted, that log isn't lost — it stays in the data repo — but it becomes undiscoverable: back on `main` you won't search a dead branch's name, and the raw checkpoints were never distilled. `handover` distils the branch's whole arc into one entry and writes it **onto `main`**, with a pointer back to the source branch so the full detail stays recoverable.

**Run this while the feature branch is still known** — ideally still checked out, before the branch is deleted.

**Not the end of a branch, just the end of a session?** Use `stop` instead. `handover` is a higher-order event: you may `stop` a branch many times, then `handover` once when it's done for good.

---

## Steps

### 1 — Determine project and branch

```bash
PROJECT=$(jotter project)
BRANCH=$(jotter branch)
```

If `BRANCH` is already `main` (or the project's trunk), stop and tell the user — there's no feature branch to hand over from. Ask them to check out the branch first, or to name it explicitly.

### 2 — Read the branch's full log

Distillation needs the whole arc, not just the tail:

```bash
jotter tail --project "$PROJECT" --branch "$BRANCH" --limit 30
```

If the branch is long-lived, widen with search rather than truncating:

```bash
jotter search --project "$PROJECT" --branch "$BRANCH" ""
```

The most recent `stop` (or legacy `finish`) entry's `**Next:**` field is a good seed for what's still outstanding.

### 3 — Distil and preview

Synthesise the branch into a handover and render it back as a quoted block **before** writing. Lead the content with a provenance line so future-you knows where it came from, then the substance:

> **Content:**
> From `<feature-branch>` (PR #NNN, merged `<sha>`).
> - <what was built, and why>
> - <key decisions / trade-offs>
> - <gotchas or debt that outlived the branch>
>
> **Next:**
> - <follow-ups that should become new work on main>

The `--next` field here isn't a session handover — the branch is dead — it's **follow-ups that outlived the branch**. If there are none, drop it.

### 4 — Write onto main — final action of the skill

Note `--branch main`: the entry lands on `main`'s log, not the (about-to-die) feature branch's.

```bash
jotter write \
  --project "$PROJECT" \
  --branch main \
  --type handover \
  --content "<content from preview>" \
  --next "<follow-ups from preview>"
```

`jotter write --type handover` commits the entry to the data repo immediately, so the handover is safe before the feature branch is deleted (deleting the branch touches the project repo, not the data repo). The remote is updated on the next background-timer push, or force it now with `jotter sync`.

### 5 — Confirm

> "Handover from `<feature-branch>` written to main's log. Safe to delete the branch — its context is committed to the data repo and recoverable from this entry."

The skill does **not** delete the branch — that's the user's call, after the handover is safely written.
