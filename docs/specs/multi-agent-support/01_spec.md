# Multi-agent support — v1 spec

## Why

Jotter's core commands (`write`, `tail`, `ls`, `search`) are already tool-agnostic — they append JSONL to a git repo and don't care which coding agent invoked them. But everything surrounding those commands is Claude Code-specific:

- The five session skills (`start`, `save`, `finish`, `break`, `recover`) ship in `SKILL.md` format only Claude Code reads
- `jotter setup` installs them to `~/.claude/skills/` and writes the `Bash(jotter:*)` permission into `~/.claude/settings.json`
- The README, cobra `Short`/`Long`, GoReleaser description, and cask caveats all say "for Claude Code sessions"

Three other coding-agent CLIs have reached feature parity on the bits jotter cares about (custom slash commands + a bash allowlist): **OpenAI Codex**, **Google Gemini CLI**, and **opencode**. Users of those tools currently can't run jotter without hand-rolling their own command files. Making jotter first-class on all four eliminates that friction and widens the addressable user base without changing the data model.

## In scope (v1, target v0.6.0)

- Ship adapted session commands for **Claude Code**, **Codex**, **Gemini CLI**, and **opencode**, covering `start` / `save` / `finish` / `break` (four of the five skills)
- Extend `jotter setup` with a "which agents do you use?" multi-select that detects installed CLIs and installs the matching commands + permission entries for each one ticked
- Re-running setup is idempotent per agent — adding a second agent later doesn't touch the first agent's install
- Update all user-facing copy (README, repo description, cobra `Short`/`Long`, GoReleaser description, cask caveats) to reflect multi-agent support
- One shared prose source for the commands — per-agent adapters differ only in frontmatter, install path, and permission wiring

## Out of scope

- **`recover` on non-Claude agents** — it reads Claude Code's JSONL transcript format from `~/.claude/projects/*.jsonl`. Codex, Gemini, and opencode each store transcripts differently (if at all); per-agent transcript parsing is a follow-up feature, not a blocker for v0.6.0. `recover` ships Claude-only; the other agents simply don't get that command installed
- **Uninstall support** — `jotter setup` is install-only today; adding per-agent uninstall is a larger change worth its own spec
- **Per-agent overrides in `.jotter` config** — a single `data_dir` still applies regardless of which agent triggered the entry
- **Aider, Cursor, Continue, any other agent not listed** — parked until there's user demand

## Target user flow

```bash
jotter setup
```

Wizard output (illustrative):

```
  ✓ Claude Code          — detected at ~/.claude
  ✓ Codex                — detected at ~/.codex
  ↷ Gemini CLI           — not installed, skipping
  ↷ opencode             — not installed, skipping

  ? Which agents should jotter wire up? [use space to toggle]
    > [x] Claude Code
      [x] Codex
```

Installing for both leaves the user with:

- `~/.claude/skills/{start,save,finish,break,recover}-session/SKILL.md`
- `~/.claude/settings.json` with `Bash(jotter:*)` in permissions
- `~/.codex/prompts/{start,save,finish,break}.md`
- (Codex has no per-command allowlist, so no permission wiring needed there)

## Acceptance criteria

- `jotter setup` on a fresh machine with all four CLIs installed offers all four in the multi-select
- Selecting a single agent installs that agent's commands and permission entries and touches no other agent's config
- Re-running `jotter setup` after ticking an additional agent adds the new agent's commands without modifying the first agent's files (StatusOK, not StatusUpdated, for the untouched one)
- Commands work end-to-end on each agent: invoking `/start` writes a start entry, `/finish` writes a finish entry, `jotter tail` shows them
- The shared prose source has no per-agent branching — only frontmatter, install path, and permission wiring differ between adapters
- Repo description, README headline, cobra `Short`/`Long`, GoReleaser description, and cask caveats all mention multi-agent support

## Non-goals for the wizard UX

- No `jotter setup --agent codex` CLI flag. One entry point, one multi-select. Fewer surfaces to document and test.
- No automatic installation of the CLIs themselves. Jotter integrates with what's already on the machine; installing Codex or Gemini is the user's problem.
