# Release infrastructure — v1 spec

## Why

Today jotter installs one way: `go install github.com/sebjacobs/jotter@latest`. That's fine for the current user (me), but it requires:

- A Go toolchain on the machine
- Willingness to trust `go install` from a public Go module
- A version scheme that's just "whatever `@latest` resolves to" — no tags, no changelog, no way to pin

The closed plugin PR (#1) tried to solve distribution by coupling it to Claude Code's marketplace. That locked jotter into one agent while still requiring Go on the machine. A plain `curl … | sh` install script backed by prebuilt binaries solves distribution cleanly and doesn't commit us to any single agent.

This spec is deliberately *only* about distribution. The `jotter setup` wizard that makes onboarding painless lives in a separate spec and ships as v0.2.0. Keeping them split means v0.1.0 can land and deliver value without waiting on the wizard.

## In scope (v0.1.0)

- Semver git tags (`v0.1.0`, `v0.2.0`, …) as the release trigger
- `CHANGELOG.md` maintained in [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format
- GoReleaser config producing per-platform archives published to GitHub Releases
- Per-platform binaries: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
- SHA-256 checksums alongside each release
- Version stamping — `jotter --version` reports semver, commit SHA, build date (via `-ldflags`)
- `install.sh` — one-line install script fetching the latest release for the user's platform
- GitHub Actions workflow triggering GoReleaser on tag push
- README updated to lead with the install script, `go install` demoted to fallback

## Out of scope

- **`jotter setup` wizard** — separate spec, target v0.2.0
- **Windows support** — no confirmed user. GoReleaser matrix excludes it; revisit when someone asks.
- **Signed binaries / notarisation** — checksums are the v0.1.0 trust baseline. Signing can come later if distribution scales.
- **Homebrew tap** — nice-to-have; GitHub Releases + install script covers the common case.
- **Auto-update** (`jotter --upgrade`) — separate feature.
- **CI other than the release workflow** — this spec adds only the release workflow; existing test CI is unchanged.

## Release process

Manual, documented in `CONTRIBUTING.md` (new, short file).

1. Move `[Unreleased]` entries in `CHANGELOG.md` under a new `[vX.Y.Z] — YYYY-MM-DD` heading; add a fresh empty `[Unreleased]` section.
2. Commit: `chore: release vX.Y.Z`.
3. Tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`.
4. Push: `git push origin main vX.Y.Z`.
5. GitHub Actions runs GoReleaser; release appears on the repo's Releases page within a minute or two.

Kept manual deliberately — auto-releasing on every merge to main would need conventional-commit parsing or release-please, which is overkill for a hobby-scale project with one maintainer.

## User-facing install flow

### Primary (new)

```bash
curl -fsSL https://raw.githubusercontent.com/sebjacobs/jotter/main/install.sh | sh
```

Script:

1. Detects OS + arch, maps to GoReleaser's archive naming
2. Calls GitHub API (`/repos/sebjacobs/jotter/releases/latest`) for the latest tag
3. Downloads the matching `jotter_<version>_<os>_<arch>.tar.gz` and its checksum file
4. Verifies SHA-256
5. Extracts `jotter` to `$HOME/.local/bin` (falls back to `/usr/local/bin` if writable)
6. Checks `$PATH`; if the target dir isn't on it, prints the one-line `export PATH=…` the user needs
7. Prints a short "installed vX.Y.Z to /path" confirmation

Unsupported OS/arch (e.g. Windows, freebsd) → print `go install` fallback and exit 1.

### Fallback (existing)

```bash
go install github.com/sebjacobs/jotter@latest
```

Stays working — this is the path Go-fluent users will keep preferring, and covers platforms not in the GoReleaser matrix.

## Version stamping

Three variables filled by `-ldflags` at build:

```go
// cmd/version.go
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)
```

- Local `go build` gets the defaults (`dev` / `none` / `unknown`)
- GoReleaser fills them on tag builds: `version` = semver tag, `commit` = full SHA, `date` = RFC3339 build time
- `jotter --version` prints all three as a small formatted block

## CHANGELOG.md — starter shape

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.1.0] — 2026-04-XX

### Added
- Prebuilt per-platform binaries published to GitHub Releases on tag push
- One-line install script (`install.sh`)
- `jotter --version` reports semver, commit SHA, and build date
- `CHANGELOG.md` (this file)

### Notes
- v0.1.0 captures the existing command surface (`write`, `tail`, `ls`, `search`, `config`, `completion`) as the first shipped baseline. Prior commits on main are folded in as "the initial feature set."
```

Subsequent releases append sections following Keep-a-Changelog headings (`Added`, `Changed`, `Fixed`, `Removed`, `Deprecated`, `Security`).

## Acceptance criteria

1. Push tag `v0.1.0` → GitHub Actions produces a release with four archives (darwin×2, linux×2) + a checksums file within a few minutes
2. `curl -fsSL …/install.sh | sh` on macOS arm64 installs `jotter` and `jotter --version` reports `v0.1.0` with a non-`none` commit and a non-`unknown` date
3. Same install flow on a fresh Linux amd64 environment works end-to-end
4. `go install github.com/sebjacobs/jotter@v0.1.0` still works as the fallback
5. README's install section leads with the one-liner, lists `go install` as the alternative

## Open questions

- **Should `install.sh` be checksummed / signed itself?** Today `curl | sh` trusts the TLS cert and the HEAD of `main`. For v0.1.0 accept this — it's the same trust model as Homebrew's install script. Revisit only if signing becomes feasible.
- **Should the install script default to `$HOME/.local/bin` or `/usr/local/bin` first?** Leaning `$HOME/.local/bin` — no sudo required, matches modern conventions (XDG, `uv`, `pnpm` install scripts). Confirm before implementation.

## Next stage

- `02_plan.md` — concrete file layout, GoReleaser config shape, GitHub Actions workflow, install.sh structure, order of work
