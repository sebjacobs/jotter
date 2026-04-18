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

### Multi-agent support

Generalise `jotter setup` beyond Claude Code to Codex, Aider, Cursor. Detect which agents are installed, offer to wire each one up with the equivalent of skills/permissions for that agent. Structured as a plugin per agent so adding a new one is additive.

**Prerequisite:** setup wizard v0.2.0 (Claude Code only) ships first and proves the flow works end-to-end. Don't generalise a flow that hasn't been stabilised.

## Shipped

- **`justfile` + CI lint fix** (v0.2.2, PR #12 merged d2a8519) — errcheck failures on `fmt.Fprint*` in setup wizard cleared; new `justfile` with `just check` running build + test + lint locally, mirroring CI. `README.md` and `CLAUDE.md` point at it as the canonical pre-push command.
- **Skip git push when data repo has no remote** (v0.2.1, PR #11 merged ad50d1b) — `finish` entries now probe for a remote via `GitHasRemote` before pushing, eliminating the `Warning: git push failed:` noise on local-only data repos. Real push failures against configured remotes still warn.
- **`jotter setup` interactive wizard** (v0.2.0, merged b880730) — seven-step wizard for Claude Code onboarding; embedded session-management skills via `//go:embed`; idempotent, always-prompt-with-current-values, accept-default is a genuine no-op.
- **`install.sh` one-line installer** (PR #8, merged 0893688) — detects OS/arch, fetches latest release, SHA-256 verifies, installs to `$HOME/.local/bin`; README rewritten to lead with it.
- **Release infrastructure + v0.1.0** (PR #7, merged 6443caf; v0.1.0 at 43d8511) — GoReleaser, CHANGELOG, CONTRIBUTING, version stamping, GitHub Actions release workflow on `v*` tag push.
- **Per-repo data dir via `.jotter` file** (merged 83e8d41) — TOML walk-up resolution replaces `JOTTER_DATA` env + `~/.config/jotter/config`. `jotter config` subcommand prints resolved data dir.
- **ASCII banner** (PR #6, merged 2b01e04) — braille otter render + figlet wordmark embedded via `//go:embed` on the root command.
