# Release infrastructure — implementation plan

Concretises `01_spec.md` into files, config, and order of work.

## Files added

```
.goreleaser.yaml                  ← release build config
.github/workflows/release.yml     ← tag-push trigger → goreleaser
install.sh                        ← one-line install script
CHANGELOG.md                      ← Keep a Changelog
CONTRIBUTING.md                   ← how to cut a release (short)
cmd/version.go                    ← version/commit/date vars + --version output
```

## Files modified

- `main.go` or `cmd/root.go` — wire `--version` output (whichever fits cobra's conventions best; likely via `rootCmd.Version`)
- `README.md` — install section rewritten: `curl … | sh` leads, `go install` is the fallback
- `.gitignore` — add `dist/` (GoReleaser's output dir)

## `.goreleaser.yaml` — shape

```yaml
version: 2

project_name: jotter

before:
  hooks:
    - go mod tidy

builds:
  - env:
      - CGO_ENABLED=0
    goos: [darwin, linux]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X github.com/sebjacobs/jotter/cmd.version={{.Version}}
      - -X github.com/sebjacobs/jotter/cmd.commit={{.FullCommit}}
      - -X github.com/sebjacobs/jotter/cmd.date={{.Date}}

archives:
  - formats: [tar.gz]
    name_template: "jotter_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - LICENSE
      - README.md
      - CHANGELOG.md

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

changelog:
  disable: true   # hand-maintained in CHANGELOG.md

release:
  header: |
    See [CHANGELOG.md](https://github.com/sebjacobs/jotter/blob/main/CHANGELOG.md) for the full list of changes.
```

Open decisions (defer to implementation):

- Whether to add a `snapshot` section for local dry-run builds (probably yes — helps test the config without cutting a real release)
- Whether `LICENSE`/`README.md`/`CHANGELOG.md` get bundled into each archive (leaning yes — costs nothing, helps users)

## `.github/workflows/release.yml` — shape

```yaml
name: release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Uses `GITHUB_TOKEN` (auto-provided by Actions) — no extra secrets.

## `cmd/version.go` — shape

```go
package cmd

import (
    "fmt"
    "runtime"
)

var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func versionString() string {
    return fmt.Sprintf("jotter %s\ncommit: %s\nbuilt:  %s\ngo:     %s",
        version, commit, date, runtime.Version())
}
```

Wire into cobra:

```go
// in root.go
rootCmd.Version = versionString()
rootCmd.SetVersionTemplate("{{.Version}}\n")
```

`jotter --version` prints the block; `jotter -v` works too (cobra wires both by default).

## `install.sh` — shape

```sh
#!/bin/sh
set -eu

REPO="sebjacobs/jotter"
INSTALL_DIR="${JOTTER_INSTALL_DIR:-$HOME/.local/bin}"

# 1. Detect OS + arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) echo "Unsupported arch: $ARCH — fall back to: go install github.com/$REPO@latest" >&2; exit 1 ;;
esac
case "$OS" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $OS — fall back to: go install github.com/$REPO@latest" >&2; exit 1 ;;
esac

# 2. Find latest tag via GitHub API
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | \
  awk -F'"' '/"tag_name":/ {print $4; exit}')

# 3. Download archive + checksums
ARCHIVE="jotter_${VERSION#v}_${OS}_${ARCH}.tar.gz"
URL_BASE="https://github.com/$REPO/releases/download/$VERSION"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL -o "$TMPDIR/$ARCHIVE" "$URL_BASE/$ARCHIVE"
curl -fsSL -o "$TMPDIR/checksums.txt" "$URL_BASE/checksums.txt"

# 4. Verify SHA-256
cd "$TMPDIR"
grep " $ARCHIVE\$" checksums.txt | sha256sum -c -

# 5. Extract + install
tar -xzf "$ARCHIVE"
mkdir -p "$INSTALL_DIR"
install -m 0755 jotter "$INSTALL_DIR/jotter"

# 6. PATH check
if ! echo ":$PATH:" | grep -q ":$INSTALL_DIR:"; then
  echo ""
  echo "Installed jotter $VERSION to $INSTALL_DIR"
  echo "Add this to your shell rc file so jotter is on PATH:"
  echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
else
  echo "Installed jotter $VERSION to $INSTALL_DIR"
fi
echo "Next: run 'jotter --version' to confirm, then see https://github.com/$REPO#setup"
```

Decisions reflected:

- Defaults to `$HOME/.local/bin` per the open question in 01_spec.md (resolving with the leaning choice)
- Overridable via `JOTTER_INSTALL_DIR` env var for CI / custom setups
- Uses `sha256sum` (Linux) / `shasum -a 256` (macOS) — worth a portability check in implementation (may need to detect and swap)

## `CONTRIBUTING.md` — shape

Short — just covers the release cut. Full text:

```markdown
# Contributing

## Cutting a release

1. Update `CHANGELOG.md`: move `[Unreleased]` entries under `[vX.Y.Z] — YYYY-MM-DD`, add a fresh empty `[Unreleased]` section.
2. Commit: `chore: release vX.Y.Z`.
3. Tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`.
4. Push: `git push origin main vX.Y.Z`.
5. Confirm the release appears at https://github.com/sebjacobs/jotter/releases within a few minutes.

Version bumps follow [Semantic Versioning](https://semver.org). Pre-1.0: minor for features, patch for fixes — no strict breaking-change guarantees.
```

## Order of work

Each step leaves main deliverable and green. Steps 1–4 can ship as separate commits on `feature/release-infra`; steps 5–6 are separate PRs (they require a merged release-infra to be useful).

1. **`CHANGELOG.md` + `CONTRIBUTING.md`** — no code changes, just the docs. Easy win, makes the release process reviewable.
2. **`cmd/version.go` + `--version` wiring** — adds version output. Local `go build` still works (gets `dev`/`none`/`unknown`). Unit test for `versionString()` formatting.
3. **`.goreleaser.yaml`** — can be validated locally with `goreleaser release --snapshot --clean` (dry-run, no tag needed). Commit once the snapshot looks right.
4. **`.github/workflows/release.yml`** — not triggerable until there's a tag, but commit so it's in place when we push v0.1.0.
5. **Raise PR, merge `feature/release-infra` to main.** Branch is done at this point.
6. **Cut v0.1.0** — follow CONTRIBUTING.md: CHANGELOG, commit, tag, push. Verify the release appears and the archives work.
7. **`install.sh`** — land after v0.1.0 exists so the script has a real release to test against. Separate small PR. README update lives in the same PR.
8. **README install section update** — lead with install script, demote `go install`. Part of the install.sh PR.

Steps 7–8 deliberately come after the tag because `install.sh` needs a release to download from — writing it against a hypothetical v0.1.0 is backwards.

## Testing strategy

- **`cmd/version.go`** — unit test that `versionString()` handles the default values correctly (no panic when `version == "dev"`) and formats consistently
- **`.goreleaser.yaml`** — `goreleaser check` in CI or locally as a lint step; `goreleaser release --snapshot --clean` for a full dry-run before tagging
- **`.github/workflows/release.yml`** — no pre-tag testing possible; first real test is pushing `v0.1.0`. Accept this — workflows are hard to test without running them.
- **`install.sh`** — manual test on macOS arm64 and Linux amd64 (VM or container) after v0.1.0 is cut. Also worth a `shellcheck` pass as part of the commit.

## Out of this plan

- Auto-update mechanism
- Homebrew tap / install methods beyond the script
- Signed binaries
- Windows / freebsd / other platforms
- Release-please or conventional-commits-based auto-changelog

All deferred to future specs if/when they become real needs.
