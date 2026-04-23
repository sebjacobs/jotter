---
name: finish-session
description: Write a finish entry to the jotter log at the end of a session, capturing what shipped and the handover for next session. Use when the user says "/finish", "/end", "let's wrap up", "wrap up", "let's finish", "end this session", "let's call it", "that's enough for today", or similar.
---

# Finish Session

Writes a `finish` entry to the jotter log — the session summary plus a `--next` handover field that `/start` will surface next time. Layer your own end-of-session conventions (commits, roadmap updates, doc refresh) on top of this skill.

**Mid-session break, not the end?** Use `break-session` instead.

---

## Steps

### 1 — Determine project and branch

```bash
PROJECT=$(jotter project)
BRANCH=$(jotter branch)
```

### 2 — Write the finish entry

Summarise the session — what was built or fixed, key decisions, gotchas or debt left behind. The `--next` field is the handover: the 2-3 most important things to pick up next session, in priority order.

```bash
jotter write \
  --project "$PROJECT" \
  --branch "$BRANCH" \
  --type finish \
  --content "<session summary: what shipped, decisions made, gotchas>" \
  --next "<top priorities for next session, in order>"
```

`jotter write --type finish` auto-commits and pushes the data repo so the handover is durable even if the machine goes away.
