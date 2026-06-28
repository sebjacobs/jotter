# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.16.1] — 2026-06-28

### Removed
- The Gatekeeper quarantine-strip cask hook is gone. It was a stopgap that ran `xattr -dr com.apple.quarantine` post-install to keep an unsigned binary from tripping Gatekeeper on first run. Now that release binaries are signed with a Developer ID Application certificate and notarized (v0.16.0), Gatekeeper accepts them on their own — `spctl -a` reports `source=Notarized Developer ID`, accepted — so the strip is redundant and the cask no longer fights Homebrew's quarantine behaviour.

## [v0.16.0] — 2026-06-28

### Changed
- macOS release binaries are now signed with a Developer ID Application certificate and notarized by Apple, via [quill](https://github.com/anchore/quill) wired into GoReleaser's `binary_signs`. The release runs on a Linux runner, so native `codesign`/`notarytool` aren't available; quill signs and notarizes Mach-O binaries from Linux without Xcode. quill errors on non-Mach-O input and the OSS `binary_signs` has no `goos` filter, so the build is split into separate darwin/linux ids and signing is scoped to the darwin id. Credentials come from five repo secrets (the Developer ID `.p12` plus its password, and an App Store Connect API key for the notary service). Snapshots sign ad-hoc and skip the notary, so `goreleaser release --snapshot` still runs credential-free. The Gatekeeper quarantine-strip cask hook stays in place for now and will be removed in a follow-up once a notarized release is confirmed passing Gatekeeper.

## [v0.15.0] — 2026-06-27

### Added
- `handover` entry type and bundled `handover` skill — distil a finished feature branch's log into a single entry written onto `main`, so its context survives the branch being deleted. A branch's log is keyed by branch name and becomes undiscoverable once the branch is merged and removed; `handover` captures the branch's arc (what was built, key decisions, surviving follow-ups) with a pointer back to the source branch. Like every entry it commits locally and is pushed asynchronously by the background timer.

### Changed
- The session-end entry type is now `stop`, renamed from `finish` (`/finish` overclaimed completion — you stop a branch for the day, you don't finish it, and `handover` now carries the "done for good" meaning). The bundled `finish-session` skill becomes `stop-session`, triggered by `/stop` and phrasings like "let's stop for today". `finish` stays a fully accepted legacy alias — old logs read fine, `jotter write --type finish` still works, and `/finish` still triggers the skill — so nothing breaks. Note: `jotter setup` installs `stop-session` alongside the now-orphaned `finish-session`; remove the old skill directory by hand if you want it gone.

## [v0.14.0] — 2026-06-27

### Added
- `jotter sync --all` — push every registered data repo that has a remote in one go. Each `jotter write` now records its resolved `data_dir` in a small global registry (`~/.jotter.d/registry`, overridable via `$JOTTER_STATE_DIR`), so the background pusher can reach all of a user's session-loggers without being handed the list. Repos without a remote are skipped; one repo's failure doesn't abort the rest, and the command only errors when every repo it attempted failed. Output is sequential, with each repo's result indented under its header and a per-run timestamp so the daemon log stays readable across runs.
- `jotter daemon install` / `uninstall` / `status` — manage a macOS launchd timer that runs `jotter sync --all` on an interval (default 300s, `--interval` to change), so writes never block on the network. `install` writes a LaunchAgent and reloads it; `status` reports whether it's installed and loaded plus a recent-log tail; `uninstall` removes it. `jotter setup` gains a step that offers to install the timer once a remote is configured. The agent's label honours `$LAUNCHD_PREFIX` (`<prefix>.jotter`, default `com.jotter`), so it slots into an existing launchd naming scheme with no jotter-specific config — purely opt-in. launchd is macOS-only; other platforms get a clear pointer to cron/systemd.

### Changed
- `finish` entries no longer push synchronously. Every write, finish included, now just commits locally and returns — pushing is asynchronous, carried by the launchd timer (`jotter daemon`) on its interval, or forced immediately with `jotter sync`. This makes writes uniformly fast and never network-bound; the trade-off is a weaker walk-away guarantee (a finish reaches the remote by the next timer tick rather than instantly).

### Fixed
- `TestGitConfigSet_SurvivesBranchRename` is no longer fragile to the host git's `init.defaultBranch`. The test helper created its repo with a bare `git init` and then renamed `main`, so on a runner defaulting to `master` (the GitHub Actions Linux image) the rename failed before the assertion. The helper now uses `git init -b main` for a deterministic branch name.

## [v0.13.0] — 2026-06-27

### Added
- Automatic branch-rename tracking. Logs are stored one file per branch, so `git branch -m` previously orphaned a branch's history under the old name. Jotter now stamps a stable id in the project repo's git config (`branch.<name>.jotter-id`, which survives the rename) plus a `<branch>.jsonl.id` sidecar on first write; the next write after a rename moves the logfile into place so history stays continuous. Best-effort and scoped to on-branch writes — scripted and cross-project writes are unaffected.
- `jotter branch mv <old> <new>` — the manual counterpart, for renaming a branch's logs when you won't write to it again. Moves `logs/<project>/<old>.jsonl` (and its sidecar) to `<new>` and commits, refusing to overwrite an existing destination.
- `jotter branch adopt` — anchor every existing local branch that has logs but isn't tracked yet, so a rename before the branch's next write is still followed. Idempotent; run once per repo to migrate pre-existing history.

## [v0.12.0] — 2026-06-27

### Changed
- `jotter write` and `jotter tail` now infer `--project` and `--branch` from git by default. When omitted, the project is the basename of the git toplevel (the same value `jotter project` prints) and the branch is the current git branch (matching `jotter branch`), so a bare `jotter write --type note --content "…"` or `jotter tail` Just Works inside a repo. Passing either flag explicitly still overrides the inferred value. Outside a git repo, the commands now error asking for the flag rather than failing a required-flag check.

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

[Unreleased]: https://github.com/sebjacobs/jotter/compare/v0.16.1...HEAD
[v0.16.1]: https://github.com/sebjacobs/jotter/releases/tag/v0.16.1
[v0.16.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.16.0
[v0.15.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.15.0
[v0.14.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.14.0
[v0.13.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.13.0
[v0.12.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.12.0
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
