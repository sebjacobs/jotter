# Jotter roadmap

Living document — Now / Next / Later priorities for jotter.

## Now

### Release infrastructure — prebuilt binaries + CHANGELOG (target: v0.1.0)

Prebuilt per-platform binaries via GoReleaser, semver tags, `CHANGELOG.md` in Keep-a-Changelog format, and a one-line install script. Delivers value on its own: anyone can `curl … | sh` and get a working `jotter` without a Go toolchain — even before `jotter setup` exists.

**Why this first:** the closed plugin PR (#1) tried to solve distribution by coupling it to Claude Code's marketplace. That locked jotter into one agent and still required Go on the user's machine. Prebuilt binaries + a plain install script solve distribution cleanly and don't commit us to any single agent. Ships as **v0.1.0**.

See `docs/specs/release-infra/`.

### `jotter setup` wizard (target: v0.2.0)

Go subcommand that takes a user from binary-on-disk to working `/start` in one flow: detects Claude Code, prompts for data dir, inits the repo, writes `.jotter`, installs embedded skills, merges `Bash(jotter:*)` permission, runs a smoke test.

**Scope:** Claude Code only for v1. Other agents (Codex, Aider, Cursor) are deferred — structure the code so they slot in later without rewrites. Depends on release-infra landing first so the wizard can ship as a proper v0.2.0 release.

See `docs/specs/setup-wizard/` (draft spec lives on `feature/setup-wizard` branch, will rebase after release-infra merges).

## Next

### Tombstone / soft-delete for entries

Entries are append-only today — no way to mark one as superseded or retract a mistake without rewriting history in the data repo (which breaks the append-only guarantee and any git-based replication).

**Target shape:** a git-style chain-of-hashes approach. Each entry gets a stable hash; a later entry can carry a `replaces: <hash>` field to mark the earlier one as superseded. Readers (`tail`, `search`) filter out replaced entries by default, with a flag to surface them.

**Why this shape:** preserves append-only semantics (nothing is ever mutated or deleted from the JSONL), survives rebase/cherry-pick in the data repo, and is trivially inspectable — the full history is still there, just annotated.

**Open questions:** hash scheme (content hash vs ULID?), whether `replaces` is a single hash or a list, how `jotter write --replace <hash>` surfaces in the CLI, how skills like `/save` and `/finish` trigger it.

Worth a short spec doc in `docs/specs/tombstone-delete/` before implementing.

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

**Prerequisite:** setup wizard v1 (Claude Code only) ships first and proves the flow works end-to-end. Don't generalise a flow that hasn't been stabilised.

## Shipped

- **Per-repo data dir via `.jotter` file** (merged 83e8d41) — TOML walk-up resolution replaces `JOTTER_DATA` env + `~/.config/jotter/config`. `jotter config` subcommand prints resolved data dir.
- **ASCII banner** (PR #6, merged 2b01e04) — braille otter render + figlet wordmark embedded via `//go:embed` on the root command.
