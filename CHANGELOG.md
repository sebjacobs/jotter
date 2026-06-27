# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.11.0] — 2026-06-27

### Added
- `jotter mv <old-project> <new-project>` — rename a project's logs. A project's jotter name is the basename of its git toplevel, so renaming the project directory on disk orphans its logs under the old name; `mv` renames `logs/<old-project>` to `logs/<new-project>` and commits the move (locally, like every entry except `finish`). Refuses to overwrite an existing destination, validates both names against path traversal, and is what `proj mv`'s log migration delegates to.

## [v0.10.0] — 2026-06-19

### Added
- `jotter sync` — recover from a failed `finish` push. When a finish entry's push fails (offline, or the remote moved on) the entry is committed locally but never reaches the remote; `jotter sync` reconciles by fetching, rebasing local entries on top of any remote commits, then pushing. It reports whether it pushed, rebased-then-pushed, or was already in sync, and errors clearly when no remote is configured.

### Changed
- `GitPush` now sets the upstream (`push -u origin HEAD`) on a branch that has no tracking branch yet, so the first push from a freshly `jotter setup` repo lands instead of failing with "no upstream branch". Bare pushes are used once tracking is established.

### Fixed
- `recover-session` no longer treats `checkpoint` and `break` entries as crash signals. The previous logic ran the full recovery flow whenever the last log line wasn't `finish`, which would wrongly close active sessions saved via `/save` + `/clear`. Recovery now only kicks in on actual crashes.

## [v0.9.0] — 2026-05-17

### Changed
- Bundled `break-session`, `finish-session`, and `save-session` skills now share an explicit walk-away guarantee — when the skill completes, dirty work is committed and the log entry is written, with no surprise follow-up actions. Each skill renders a preview quoted block before the `jotter write` call so the content is visible before it lands in the log, and ends with an explicit confirmation line stating the skill is done. The write is deliberately the final action, so a user can invoke `/finish`, see the confirmation, and close the laptop. Per-skill specialisation: `/break` uses a single auto-WIP commit (no ceremony — the user is walking away now); `/save` and `/finish` propose commit groupings and wait for approval.
- `save-session` drops its pre-read step (previously `jotter tail --limit 3` before every save). A duplicate checkpoint is cheaper than two extra reads on every invocation.
- `recover-session` no longer embeds a ~30-line python heredoc inline. The JSONL transcript filter has been extracted into `skills/recover-session/scripts/transcript.py` with `list` and `extract` subcommands. The setup wizard's `fs.WalkDir` already recurses into subdirectories, so the script installs at `~/.claude/skills/recover-session/scripts/` automatically with no code changes.

### Added
- `save-session` gains a "note" mode — single-observation jots ("make a note", "jot it down", "note that") that skip the commit step and the `--next` handover. Distinct from the heavier checkpoint mode.

## [v0.8.0] — 2026-05-17

### Added
- `jotter tail -n` — shorthand alias for `--limit`. Matches the `-n` convention used by `tail`, `head`, and `git log` for "show this many entries". `--limit` keeps working unchanged so existing scripts and skills are unaffected.

## [v0.7.2] — 2026-05-07

### Fixed
- `jotter project` now returns the main repo's name when run from inside a git worktree. Previously it called `git rev-parse --show-toplevel`, which inside a worktree returns the worktree's checkout dir — so a worktree at `<repo>/.claude/worktrees/feature-x` would report `feature-x` as the project name, splitting that project's session log across multiple synthetic "projects" depending on whether work happened in the main checkout or a worktree. Now uses `git rev-parse --path-format=absolute --git-common-dir` and takes the parent's basename — `--git-common-dir` always points at the main repo's `.git` even from inside a worktree, so the result is the project name regardless of where the user is checked out. Bundled session skills that call `$(jotter project)` benefit automatically. Requires git ≥ 2.31 (March 2021).

## [v0.7.1] — 2026-04-20

### Fixed
- Release-infrastructure patch. v0.7.0's release workflow uploaded assets successfully but failed at the Homebrew tap step with a `403` on `HOMEBREW_TAP_GITHUB_TOKEN`, leaving `sebjacobs/homebrew-tap/Casks/jotter.rb` pinned at v0.5.0. Re-running the v0.7.0 workflow hit `422 already_exists` on asset upload, so this patch version exists purely to re-trigger the release workflow with a rotated PAT and push the cask forward. No code changes.

## [v0.7.0] — 2026-04-19

### Added
- `jotter ls --since` / `--until` — filter the project, branch, or entry list to a date/timestamp window. Mirrors the flags just added to `jotter search` in v0.6.0: same formats (`YYYY-MM-DD` or `YYYY-MM-DDTHH:MM:SS`), same inclusive semantics on both ends. Makes "which projects did I touch on date X?" a one-liner (`jotter ls --since X --until X`). `last:` timestamps and entry counts reflect the in-window slice so the display stays internally consistent — no overall-last timestamps leaking into a filtered view.

### Changed
- Extracted `parseBoundary` / `parseWindow` / `inWindow` into `cmd/boundary.go` so `ls` and `search` share one parser. Pure refactor — behavioural parity with v0.6.0.

## [v0.6.0] — 2026-04-19

### Added
- `jotter search --until` — new upper-bound flag that pairs with the existing `--since` to scope results to a date range. When both bounds match (`--since X --until X`), every entry from that single day is returned — the common case when reviewing one session's log. Both `--since` and `--until` also accept full `YYYY-MM-DDTHH:MM:SS` timestamps, so windows can be as tight as needed (e.g. isolating a single morning). Date values remain inclusive on both ends so there's no half-open interval to reason about.

## [v0.5.0] — 2026-04-19

### Added
- `jotter project` and `jotter branch` — two tiny helper subcommands that print the current project name (basename of the git toplevel) and current branch. Intended for use in skill templates and scripts that previously boilerplated `basename "$(git rev-parse --show-toplevel)"` and `git rev-parse --abbrev-ref HEAD` everywhere. Error out cleanly outside a git repo or on detached HEAD (for `branch`). `write` / `tail` / `ls` still require explicit `--project` / `--branch` flags — no behavioural magic on the hot path; see ROADMAP.md for the auto-detect alternative.

### Changed
- Bundled template session skills (`start-session`, `save-session`, `finish-session`, `break-session`, `recover-session`) now call `$(jotter project)` and `$(jotter branch)` instead of shelling out to raw git plumbing. Fewer lines, one mental model across all five skills.

## [v0.4.0] — 2026-04-19

### Changed
- `jotter setup`'s skills step now prompts before overwriting an existing `~/.claude/skills/<name>/SKILL.md` whose content differs from the bundled template. Previously the step overwrote local edits silently on every re-run, clobbering any customisation. Default is No, mirroring the pattern already used for `.jotter`. Byte-identical re-runs and fresh installs behave exactly as before. The summary line now reports `installed / updated / kept` so the outcome is explicit.
- Bundled template session skills (`start-session`, `save-session`, `finish-session`, `break-session`) trimmed back to their jotter-specific behaviour — determining project/branch, calling `tail` / `ls` / `write` with the right `--type` and fields, surfacing the `**Next:**` handover. Session-management conventions that had accumulated in the templates (7PM cutoff language, cron pacing rules, ROADMAP.md / DONE.md workflow, `gh pr list` TODO updates, commit-grouping proposals) have been removed — these belong in personal customisation, not the tool's bundled defaults. 406 → 270 lines total. `recover-session` was already jotter-focused and is essentially unchanged.

## [v0.3.2] — 2026-04-18

### Added
- `install.sh` now offers to run `jotter setup` immediately after installing the binary. Shrinks time-to-first-use — one `curl | sh` can take a new machine from nothing to a fully-wired Claude Code session (data repo initialised, skills installed, permissions granted, smoke test passed). Reads y/N from `/dev/tty` so the prompt works under `curl | sh` where stdin is a pipe; skipped entirely when no tty is attached so CI/docker installs don't hang. Defaults to No.

## [v0.3.1] — 2026-04-18

### Changed
- `jotter ls --project P --branch B` now lists entries newest first, matching the descending order of `ls` and `ls --project`. All three `ls` variants now share a single mental model — newest at the top. `tail` is unchanged and still prints oldest-of-tail first (standard Unix semantics).
- `jotter ls` and `jotter ls --project P` now show `HH:MM` alongside the date in the `last:` field, so recency is obvious at a glance without a follow-up `tail`.

## [v0.3.0] — 2026-04-18

### Added
- `jotter ls --project P --branch B` — new third mode that lists every entry in a branch's session log as a one-liner (timestamp, type, short title). Makes `ls` a consistent drill-down: projects → branches → entries. Scanning a branch's full history no longer requires `jotter tail --limit N`.

## [v0.2.3] — 2026-04-17

### Added
- ASCII banner now prints at the start of `jotter setup` and at the top of `install.sh`, matching the otter banner already shown by `jotter --help`. All three first-touch surfaces share the same visual identity.

### Fixed
- `--version` / `-v` now works on every subcommand (`jotter setup --version`, `jotter tail --version`, etc.), not just the root command. Reimplemented as a persistent flag + `PersistentPreRun` rather than cobra's root-only auto-registered flag.

## [v0.2.2] — 2026-04-17

### Fixed
- CI lint job is green again. `fmt.Fprintln`/`Fprintf` calls in `cmd/setup.go` and `internal/setup/wizard.go` now explicitly discard their return values, clearing `errcheck` failures that had been red since the v0.2.0 setup-wizard merge.

### Added
- `justfile` with `build`, `test`, `lint`, `check`, `release-snapshot`, and `clean` recipes. `just check` runs build + test + lint in one go, mirroring exactly what CI runs. `README.md` and `CLAUDE.md` now point at it as the canonical pre-push command.

## [v0.2.1] — 2026-04-17

### Fixed
- `finish` entries no longer print `Warning: git push failed:` when the data repo has no git remote configured. Jotter now probes for a remote before pushing and skips the push silently when none is set. Real push failures (network, auth) against a configured remote still surface as a warning.

## [v0.2.0] — 2026-04-17

### Added
- `jotter setup` — interactive wizard that takes a user from binary-installed to `/start` works in one flow: detects Claude Code, prompts for a data directory, initialises the git-backed data repo, optionally wires a git remote, writes `~/.jotter`, installs embedded session-management skills, merges the `Bash(jotter:*)` permission into `~/.claude/settings.json`, and runs a smoke test (with cleanup). Always prompts with current values as defaults — accepting every default is a genuine no-op (zero file writes, zero data-repo commits).
- Five session-management skills (`start-session`, `save-session`, `finish-session`, `break-session`, `recover-session`) embedded into the binary via `//go:embed` and installed by `jotter setup`.
- Development section in README covering build and test for external contributors not using Claude Code.

## [v0.1.0] — 2026-04-17

First tagged release. Captures the existing command surface as the baseline and introduces prebuilt per-platform binaries.

### Added
- Prebuilt per-platform binaries (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64) published to GitHub Releases on tag push, via GoReleaser.
- SHA-256 checksums alongside each release.
- `jotter --version` reports semver, commit SHA, and build date.
- `CHANGELOG.md` (this file) and `CONTRIBUTING.md` documenting the release process.
- Existing command surface — `write`, `tail`, `ls`, `search`, `config`, `completion` — folded in as the initial shipped feature set.

[Unreleased]: https://github.com/sebjacobs/jotter/compare/v0.11.0...HEAD
[v0.11.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.11.0
[v0.10.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.10.0
[v0.9.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.9.0
[v0.8.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.8.0
[v0.7.2]: https://github.com/sebjacobs/jotter/releases/tag/v0.7.2
[v0.7.1]: https://github.com/sebjacobs/jotter/releases/tag/v0.7.1
[v0.7.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.7.0
[v0.6.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.6.0
[v0.5.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.5.0
[v0.4.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.4.0
[v0.3.2]: https://github.com/sebjacobs/jotter/releases/tag/v0.3.2
[v0.3.1]: https://github.com/sebjacobs/jotter/releases/tag/v0.3.1
[v0.3.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.3.0
[v0.2.3]: https://github.com/sebjacobs/jotter/releases/tag/v0.2.3
[v0.2.2]: https://github.com/sebjacobs/jotter/releases/tag/v0.2.2
[v0.2.1]: https://github.com/sebjacobs/jotter/releases/tag/v0.2.1
[v0.2.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.2.0
[v0.1.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.1.0
