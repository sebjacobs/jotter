---
name: start-session
description: Run the session start routine — ask about available time and hard stops, read recent session logs and ROADMAP.md priorities, propose a realistic goal. Use when the user says "/start", "/start-session", "let's start", "start session", "begin session", or at the start of any longer session.
---

# Start Session

Ensures every session begins with a shared understanding of available time, current priorities, and a single concrete goal.

---

## Steps

### 0 — Get the current time

Run `date` to get the actual current time. Use this to:
- Confirm the correct date for labelling
- Compute cron expressions for hard stop warnings
- Check whether the session start is already past 7PM

### 1 — Ask the two questions

Before reading anything, ask:

> "How much time do we have today, and any hard stops during the session?"

Wait for the answer. A 30-minute session gets one small task; a 2-hour session can tackle the next sprint item.

**If the user mentions a hard stop**, schedule a one-shot warning 15 minutes before using CronCreate.

### 2 — Restore context from session logs

Determine the project name and branch:

```bash
basename "$(git rev-parse --show-toplevel)"
git rev-parse --abbrev-ref HEAD
```

Read the last few entries to understand where things left off:

```bash
jotter tail --project <project> --branch <branch> --limit 5
```

If no entries exist for this branch, check if there's context from the project's main branch:

```bash
jotter tail --project <project> --branch main --limit 3
```

Surface the most recent finish entry's `**Next:**` field — that's the handover from last session.

### 3 — Read the roadmap

```
ROADMAP.md     — Now / Next / Later priorities
```

Extract:
- The **Now** priorities
- The first **Next** item in line
- Any open items from the previous session that weren't completed

### 4 — Check open PRs

```bash
gh pr list --state open
```

Display as a table. For each open PR, check for a `## TODO before merge` checklist and surface outstanding items.

### 5 — Propose a goal

Based on available time and the roadmap, propose **one concrete thing to finish** — not a wish list.

> "Given [X time], I'd suggest we tackle **[specific task]** today — [why it's the right pick]. That should be completable in [Y time] leaving [Z buffer].
>
> Anything you want to adjust, or shall we go?"

Rules:
- One goal. Not two, not "we could also...".
- If the top item is too large, scope it down or suggest a smaller win.
- Flag any hard stops that would interrupt the work.

### 6 — Set the pacing

**30 minutes or shorter:** schedule a one-shot end warning ~5 minutes before the end.

**Longer than 30 minutes:** schedule a recurring 30-minute check-in:
- `cron`: `*/30 * * * *`
- `prompt`: `30-minute check-in — how's progress? On track for the session goal?`
- `recurring`: `true`

If past **7PM**, say so directly.

### 7 — Write the start entry

```bash
jotter write \
  --project <project> \
  --branch <branch> \
  --type start \
  --content "<session goal, available time, approach>"
```
