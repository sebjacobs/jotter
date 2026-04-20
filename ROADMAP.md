# Jotter roadmap

Living document — Now / Next / Later priorities for jotter.

## Now

### Relative timestamps in `jotter ls` (PR #10, draft)

Show human-friendly relative times ("2h ago", "yesterday") in `jotter ls` output. Branch `feature/ls-relative-timestamps` is rebased onto main and green; ready to move out of draft once the UX is confirmed.

## Next

### All session entries should land on the starting branch, not the current branch

Today every skill (`/start`, `/save`, `/break`, `/finish`) resolves the branch via `git rev-parse --abbrev-ref HEAD` at call time. That works while the branch sticks around, but fragments the session log whenever a branch is merged, renamed, or hopped mid-session: `/start` lands on `feature+foo.jsonl`, then after merging and cutting a release you run `/save` or `/finish` from `main`, and subsequent entries land on `main.jsonl`. The narrative is split across two files.

**Target shape:** the session is identified by the branch where `/start` ran. Every follow-up skill (`/save`, `/break`, `/finish`) should pass `--branch <starting-branch>` explicitly rather than re-resolving from cwd — all four skills follow the same rule for consistency. `jotter write --branch` already exists as a required flag, so the fix is purely in the skill templates in `skills/` (which `jotter setup` installs).

**Two design options — pick one:**
1. **Look backward at finish time.** `/finish` queries jotter for the most recent `start` entry across all branches for this project that has no matching `finish`, uses that branch. Stateless but adds a discovery step. Needs a way to search across branches — `jotter search --type start --project <project>` might be enough.
2. **Stash at start time.** `/start` writes the starting branch to a session-local file (e.g. `$JOTTER_DATA/.active-session`), `/finish` reads it and clears it. Simpler at finish, but adds state the user can't see and that needs cleanup on abandoned sessions.

Option 1 feels more aligned with "append-only JSONL is the source of truth" — nothing to clean up, no hidden state. It also happens to be the natural fit for `/recover`, which can't inherit a starting branch (it's trying to *find* a lost session) and needs to scan across branches anyway. Once the discovery mechanism exists for `/recover`, the other skills can reuse it. Worth a short spec in `docs/specs/session-branch-continuity/` before changing the skills.

**`/recover` is the awkward one:** run from anywhere, possibly post-crash, possibly days later. It needs to (1) discover the most recent unfinished session for this project across all branches, (2) confirm with the user which one to recover, and (3) write its recovery entry to that branch — not to cwd's branch. The discovery step is what makes option 1 the right call for all four skills.

**Trade-offs:** the log branch decouples from the cwd branch — slightly more magic, but that's the point. Post-merge releases, branch cleanups, and worktree hops no longer fragment the session.

### `jotter project` lifecycle subcommands (mv, rm, info, path)

Project name today = `filepath.Base(git rev-parse --show-toplevel)` (`internal/git.go:13`). No config override, no remote fallback. Consequences: renaming a checkout directory silently orphans prior logs under the old name; two projects sharing a basename in different parents collide; `git branch -m` fragments branch-level history the same way one level down.

**Target shape — MVP set:**
- `jotter project mv <old> <new>` — `git mv logs/<old> logs/<new>` in the data repo + commit. Symmetric `jotter branch mv <old> <new> [--project P]` for the branch-rename case.
- `jotter project rm <name>` — delete a project's logs with confirmation + git commit. Guard with `--force` / interactive confirm.
- `jotter project info [name]` — branches, entry counts per branch, first/last dates, total size. Useful pre-`mv`/`rm`.
- `jotter project path [name]` — prints `$JOTTER_DATA/logs/<name>`; scriptable (`cd "$(jotter project path)"`).

**Paired config lever:** optional `project_name` field in `.jotter` TOML. Resolution precedence becomes flag → TOML → git-basename. Decouples project identity from directory name, incidentally fixes the basename-collision case. Compose: run `project mv` to relocate logs, drop a `project_name` in `.jotter` so future sessions keep resolving to the same name after a rename. Open question: should `project mv` auto-write the `.jotter` override when it detects cwd would now resolve differently? "Do the right thing" path but couples two features.

**Skip for now:** `project ls` (overlaps with top-level `jotter ls`; revisit if metadata-summary need emerges), `project merge` (niche; wait for demand), `archive` / `export` / `open` (data-repo git history + shell already cover these).

Worth a short spec in `docs/specs/project-lifecycle/` before building — decide `mv` auto-config behaviour and whether `info`/`path` take an optional positional or default to cwd-resolved.

### Tombstone / soft-delete for entries

Entries are append-only today — no way to mark one as superseded or retract a mistake without rewriting history in the data repo (which breaks the append-only guarantee and any git-based replication).

**Target shape:** a git-style chain-of-hashes approach. Each entry gets a stable hash; a later entry can carry a `replaces: <hash>` field to mark the earlier one as superseded. Readers (`tail`, `search`) filter out replaced entries by default, with a flag to surface them.

**Why this shape:** preserves append-only semantics (nothing is ever mutated or deleted from the JSONL), survives rebase/cherry-pick in the data repo, and is trivially inspectable — the full history is still there, just annotated.

**Open questions:** hash scheme (content hash vs ULID?), whether `replaces` is a single hash or a list, how `jotter write --replace <hash>` surfaces in the CLI, how skills like `/save` and `/finish` trigger it.

Worth a short spec doc in `docs/specs/tombstone-delete/` before implementing.

### `jotter setup --update-skills` (triggered when we actually update a skill)

Today `jotter setup` does byte-compare of installed skills against the embedded version and overwrites silently on mismatch. That works technically but has two gaps: (1) users have no signal that re-running setup is needed after a binary upgrade, and (2) any local customisations to a skill get clobbered without warning.

**Target shape:** a skills-only subcommand — `jotter setup --update-skills` — that users run deliberately when upgrading. Prompts before overwriting any skill whose installed bytes differ from both the new embedded version AND the originally-installed version (indicating user customisation).

**Deferred until needed:** not worth building until we actually ship a skill update. When that happens, this is the design to reach for. Likely paired with a `.jotter-skills-state.json` file recording per-file hashes of what was originally installed, for three-way-merge customisation detection.

## Later

### Pluggable storage backends

Today jotter writes JSONL files directly and commits them to a git-backed data dir. That choice couples three concerns: local file layout, query mechanism, and replication. Splitting them would let others plug in alternative backends.

**Target shape:** a `Storage` interface with `Append(Entry)`, `Tail(project, branch, limit)`, `List(project)`, `Search(filters)`. The current JSONL+git implementation becomes one concrete type; callers in `cmd/` depend only on the interface.

**Candidates worth supporting:**

- **SQLite (local)** — cheapest second backend to prove the interface with. Drops per-write git overhead, gains indexed search, keeps local-first. Open question: whether to retain a git sync layer on top or drop it.
- **D1 (remote SQLite)** — same API as SQLite over HTTP. Gains multi-device sync, loses local-first; network dependency on every write.
- **Kafka** — **not a peer backend**. Append-friendly but not queryable (`Tail`, `--since`, `--type` all become stream-and-filter or require a companion read model). Better modelled as an export target: primary store + `jotter export kafka` or a sidecar shipper.

**Prep work that pays off regardless of whether this lands:**

1. Audit `cmd/` to ensure nothing touches `os` / `filepath` directly — everything should route through `internal`. Mostly true today; worth confirming.
2. Move `internal.GitCommit` / `GitPush` calls out of `cmd/write.go` and into the storage layer. Any non-local backend will not want per-write git commits, and this coupling will leak through any extracted interface if left alone.

**When to actually do it:** only once a concrete second backend is committed to. Extracting an interface with one implementation tends to encode the current impl's shape (per-write git commits, branch-name sanitisation as a filename concern) rather than a genuinely portable contract.

### Auto-detect `--project` / `--branch` on `write`, `tail`, `ls`

Today `write`, `tail`, and `ls` all require `--project` and `--branch` flags. The v0.5.0 `jotter project` / `jotter branch` helper commands took one step toward fixing this — skills now call them rather than boilerplating `git rev-parse` — but every skill still passes the resolved values explicitly on every call.

**Target shape:** `--project` and `--branch` become optional. When unset on `write` / `tail` / `ls`, jotter auto-detects from cwd (same logic as the `project` / `branch` subcommands: basename of git toplevel; current branch; error on detached HEAD or outside a repo). Explicit flags still win for cross-project writes and scripted flows.

**Trade-off:** more magic, less typing. The current explicit shape was chosen deliberately in v0.5.0 — callers see exactly what values are being passed. The auto-detect form has no such visibility; a `jotter write --type finish --content "…"` run from the wrong directory would silently write to a different project/branch than intended.

**When to revisit:** if skill templates keep accumulating `$(jotter project)` / `$(jotter branch)` boilerplate and the "surprise a user could write to the wrong place" risk feels overstated in practice. Would likely collapse another 20–30 lines of template bash.

### Multi-agent support

Generalise `jotter setup` beyond Claude Code to Codex, Aider, Cursor. Detect which agents are installed, offer to wire each one up with the equivalent of skills/permissions for that agent. Structured as a plugin per agent so adding a new one is additive.

**Prerequisite:** setup wizard v0.2.0 (Claude Code only) ships first and proves the flow works end-to-end. Don't generalise a flow that hasn't been stabilised.

## Shipped

- **v0.7.1 release-infra patch + Homebrew tap unblocked** (v0.7.1, tag 847a857) — rotated `HOMEBREW_TAP_GITHUB_TOKEN` (fine-grained PAT, `contents: write` on `sebjacobs/homebrew-tap`), cut a zero-code v0.7.1 to re-trigger the release workflow. `sebjacobs/homebrew-tap/Casks/jotter.rb` is now at 0.7.1, caught up from v0.5.0. Alternative (delete v0.7.0 release assets + rerun) rejected as destructive on public state. Lesson captured: `just check` and `goreleaser release --snapshot --clean` belong in the release checklist even for zero-code patches — skipped both pre-push on v0.7.1 and only got away with it because the change was CHANGELOG-only.
- **`jotter ls --since` / `--until`** (v0.7.0, PR #24 merged 6877e56) — filter the project/branch/entry list to a date or timestamp window, mirroring the flags on `search`. `last:` timestamps and counts reflect the in-window slice so the display stays internally consistent. Factored `parseBoundary` / `parseWindow` / `inWindow` into `cmd/boundary.go` as a shared helper in a standalone refactor commit.
- **`jotter search --until` + timestamp support** (v0.6.0, PR #23 merged 2c5d3b5) — new `--until` flag paired with `--since`; both now accept `YYYY-MM-DD` (inclusive on both ends) or `YYYY-MM-DDTHH:MM:SS` (exact). Single-day queries and arbitrary windows work from one flag pair.
- **`justfile` + CI lint fix** (v0.2.2, PR #12 merged d2a8519) — errcheck failures on `fmt.Fprint*` in setup wizard cleared; new `justfile` with `just check` running build + test + lint locally, mirroring CI. `README.md` and `CLAUDE.md` point at it as the canonical pre-push command.
- **Skip git push when data repo has no remote** (v0.2.1, PR #11 merged ad50d1b) — `finish` entries now probe for a remote via `GitHasRemote` before pushing, eliminating the `Warning: git push failed:` noise on local-only data repos. Real push failures against configured remotes still warn.
- **`jotter setup` interactive wizard** (v0.2.0, merged b880730) — seven-step wizard for Claude Code onboarding; embedded session-management skills via `//go:embed`; idempotent, always-prompt-with-current-values, accept-default is a genuine no-op.
- **`install.sh` one-line installer** (PR #8, merged 0893688) — detects OS/arch, fetches latest release, SHA-256 verifies, installs to `$HOME/.local/bin`; README rewritten to lead with it.
- **Release infrastructure + v0.1.0** (PR #7, merged 6443caf; v0.1.0 at 43d8511) — GoReleaser, CHANGELOG, CONTRIBUTING, version stamping, GitHub Actions release workflow on `v*` tag push.
- **Per-repo data dir via `.jotter` file** (merged 83e8d41) — TOML walk-up resolution replaces `JOTTER_DATA` env + `~/.config/jotter/config`. `jotter config` subcommand prints resolved data dir.
- **ASCII banner** (PR #6, merged 2b01e04) — braille otter render + figlet wordmark embedded via `//go:embed` on the root command.
