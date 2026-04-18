# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/sebjacobs/jotter/compare/v0.3.2...HEAD
[v0.3.2]: https://github.com/sebjacobs/jotter/releases/tag/v0.3.2
[v0.3.1]: https://github.com/sebjacobs/jotter/releases/tag/v0.3.1
[v0.3.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.3.0
[v0.2.3]: https://github.com/sebjacobs/jotter/releases/tag/v0.2.3
[v0.2.2]: https://github.com/sebjacobs/jotter/releases/tag/v0.2.2
[v0.2.1]: https://github.com/sebjacobs/jotter/releases/tag/v0.2.1
[v0.2.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.2.0
[v0.1.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.1.0
