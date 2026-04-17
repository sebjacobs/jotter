# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `CHANGELOG.md` (this file) and `CONTRIBUTING.md` documenting the release process.

## [v0.1.0] — TBD

First tagged release. Captures the existing command surface as the baseline and introduces prebuilt per-platform binaries.

### Added
- Prebuilt per-platform binaries (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64) published to GitHub Releases on tag push, via GoReleaser.
- SHA-256 checksums alongside each release.
- One-line install script: `curl -fsSL https://raw.githubusercontent.com/sebjacobs/jotter/main/install.sh | sh`.
- `jotter --version` reports semver, commit SHA, and build date.
- Existing command surface — `write`, `tail`, `ls`, `search`, `config`, `completion` — folded in as the initial shipped feature set.

[Unreleased]: https://github.com/sebjacobs/jotter/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/sebjacobs/jotter/releases/tag/v0.1.0
