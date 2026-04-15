---
name: finish-session
description: Run the end-of-session checklist — write session summary, move completed items to DONE.md, add new items to ROADMAP.md, review dirty git state and propose commits. Use when the user says "/finish", "/end", "let's wrap up", "wrap up", "let's finish", "end this session", "let's call it", "that's enough for today", or similar.
---

# Finish Session

Runs the end-of-session checklist. Ensures every session ends cleanly with the handoff state captured for next time.

**Mid-session break (`/break`)?** Use the `break-session` skill instead.

---

## Steps

### 0 — Get context

Run `date` to get the actual current time. Check whether the session is running past 7PM — if so, flag it.

Determine the project name and branch:

```bash
basename "$(git rev-parse --show-toplevel)"
git rev-parse --abbrev-ref HEAD
```

### 1 — Write the finish entry

Summarise the session — what was built or fixed, key decisions, anything discovered that changed the plan. The `--next` field is the handover: the 2-3 most important things to pick up next session, in priority order.

```bash
jotter write \
  --project <project> \
  --branch <branch> \
  --type finish \
  --content "<session summary: what shipped, decisions made, gotchas/debt>" \
  --next "<top priorities for next session, in order>"
```

This auto-commits and pushes to the data repo remote.

### 2 — Move completed items to DONE.md

Scan `ROADMAP.md` for any completed items. Move them to the top of `DONE.md` under today's date heading. Remove them from `ROADMAP.md`.

### 3 — Add new items

Anything discovered during the session that needs doing:
- Active sprint task → `ROADMAP.md` **Now**
- Agreed next priority → `ROADMAP.md` **Next**
- Backlog idea → `ROADMAP.md` **Later** (or `BACKLOG.md` if detailed)

### 4 — Update roadmap horizons

In `ROADMAP.md`:
- If a **Next** item was started this session, move it to **Now**
- If priorities shifted, reorder accordingly
- If a **Later** item is now ready, check it has a spec before moving to **Next**

### 5 — Update CLAUDE.md

If anything changed — new script, renamed column, updated workflow, schema change — update the relevant section of the project's `CLAUDE.md`. Future sessions start by reading it; stale docs are worse than no docs.

### 6 — Update open PR TODO checklists

```bash
gh pr list --state open
```

For each open PR, review what's left and update the `## TODO before merge` checklist in the PR description.

### 7 — Check dirty state and propose commits

```bash
git status
git diff --stat
```

Survey all uncommitted changes. Propose a grouping to the user — one commit per logical change. Wait for approval before committing.

### 8 — Final check

```bash
git status
```

Tree should be clean. If not, flag remaining files and ask whether to commit, stash, or leave. Confirm push if not already done.

Cancel the session cron timer if one is running (`CronDelete <job-id>`).

---

## Sign-off

> "Done. Today: [what shipped]. Next session: [top priority]."
