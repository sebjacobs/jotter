---
name: recover-session
description: Recover context from a crashed or unfinished session by reading the most recent JSONL transcript. Use when the user says "/recover", "recover session", "what was I doing", or when /start detects the last entry isn't a finish.
---

# Recover Session

Reconstructs what happened in a session that ended without `/finish` — reads Claude Code's JSONL transcript and writes a recovery entry to the session log.

---

## Steps

### 0 — Detect the situation

Determine the project name and branch:

```bash
PROJECT=$(jotter project)
BRANCH=$(jotter branch)
```

Check the last session log entry:

```bash
jotter tail --project "$PROJECT" --branch "$BRANCH" --limit 1
```

If the last entry is a `finish`, the session ended cleanly — nothing to recover. Tell the user and stop.

If the last entry is a `checkpoint` or `break`, the session was intentionally paused — `checkpoint` via `/save` (typically before `/clear` to trim context), `break` via `/break` (stepping away briefly). Neither is a crash. Show the entry to the user and stop; they can continue the session normally. Only proceed if the user explicitly says the session crashed or if the entry is clearly stale (e.g. >24h old with no further activity).

If the last entry is a `start` with no follow-up, or there are no entries at all, the previous session likely crashed before any checkpoint was written — proceed to step 1.

### 1 — Find the transcript

This skill ships a helper at `scripts/transcript.py` that lists recent JSONLs and extracts human/assistant turns. Use it rather than re-deriving the project-dir path or re-implementing the filter inline.

```bash
~/.claude/skills/recover-session/scripts/transcript.py list
```

Show the user the most recent entry (timestamp + first user message) and ask: "Is this the session to recover from, or should I look at an older one?" Wait for confirmation.

### 2 — Extract the conversation

```bash
~/.claude/skills/recover-session/scripts/transcript.py extract <path-from-step-1>
# writes /tmp/recovered-session.md by default; pass --out to override
```

Read `/tmp/recovered-session.md` to understand what happened.

### 3 — Write the recovery entry

From the extracted conversation, synthesise a recovery entry:

```bash
jotter write \
  --project "$PROJECT" \
  --branch "$BRANCH" \
  --type finish \
  --content "<what was built/fixed, key decisions, where things stopped>" \
  --next "<priorities inferred from the session trajectory>"
```

Use `--type finish` so the log correctly marks the session as closed.

### 4 — Report

> "Recovered session from [date/time]. Key context: [1-2 sentence summary]."

Flag anything that couldn't be recovered (very short session, mostly tool output).
