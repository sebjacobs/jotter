# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `jotter setup` — interactive wizard that takes a user from binary-installed to `/start` works in one flow: detects Claude Code, prompts for a data directory, initialises the git-backed data repo, optionally wires a git remote, writes `~/.jotter`, installs embedded session-management skills, merges the `Bash(jotter:*)` permission into `~/.claude/settings.json`, and runs a smoke test. Idempotent — re-running updates only what's changed.
- Five session-management skills (`start-session`, `save-session`, `finish-session`, `break-session`, `recover-session`) embedded into the binary via `//go:embed` and installed by `jotter setup`.

## [v0.1.0] — 2026-04-17

First tagged release. Captures the existing command surface as the baseline and introduces prebuilt per-platform binaries.

### Added
- Prebuilt per-platform binaries (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64) published to GitHub Releases on tag push, via GoReleaser.
- SHA-256 checksums alongside each release.
- `jotter --version` reports semver, commit SHA, and build date.
- `CHANGELOG.md` (this file) and `CONTRIBUTING.md` documenting the release process.
- Existing command surface — `write`, `tail`, `ls`, `search`, `config`, `completion` — folded in as the initial shipped feature set.

[Unreleased]: https://github.com/sebjacobs/jotter/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.1.0
