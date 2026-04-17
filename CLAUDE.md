# Jotter

Append-only session log tool for Claude Code sessions. Go rewrite of the Python `session_logger.py`.

## Build & test

```bash
go build -o bin/jotter .     # build binary (always output to bin/, not repo root)
go test ./...                # run all tests (61 tests)
go test ./cmd/               # command-level integration tests
go test ./internal/          # unit tests for config, entry, storage
```

Tests build the binary once via `TestMain` and run it as a subprocess with a temp git-backed data dir. No mocks.

Local builds get placeholder version info (`dev` / `none` / `unknown`). Release builds via GoReleaser fill in real semver, commit SHA, and build date via `-ldflags`.

## Architecture

```
main.go              -> cmd.Execute()
cmd/
  root.go            -> cobra root command, --version wiring
  banner.txt         -> ASCII banner embedded into root command Long description
  version.go         -> version/commit/date vars (ldflags-stamped) + formatter
  write.go           -> append JSONL entry + git commit (+ push on finish)
  tail.go            -> read last N entries, render as markdown
  ls.go               -> list projects/branches with metadata
  search.go          -> filter entries by term, type, date, scope
  config.go          -> print resolved .jotter data_dir for current cwd
  completion.go      -> bash/zsh/fish completion generator
internal/
  config.go          -> resolve data dir by walking up from cwd for .jotter TOML files; falls back to ~/.jotter
  entry.go           -> Entry struct, JSONL marshal (Python-compatible spacing), markdown format
  storage.go         -> path construction, branch sanitisation (/ -> +), glob collection
  git.go             -> git add/commit/push via exec.Command
  color.go           -> TTY-aware ANSI colouring helpers
```

## Key conventions

- JSONL uses Python `json.dumps` spacing (`, ` and `: ` separators) for data repo compatibility
- Branch names sanitised: `/` replaced with `+` in filenames
- Entry types: `start`, `checkpoint`, `note`, `break`, `finish`
- Git commit message format: `session: {project}/{branch} {type} {timestamp}`
- `finish` entries trigger git push (non-fatal on failure)
- Exit code 1 for user-facing errors (missing files, no results, invalid input)

## Release

Prebuilt per-platform binaries are published to GitHub Releases via GoReleaser on `v*` tag push (`.github/workflows/release.yml`). See `CONTRIBUTING.md` for the five-step release cut procedure. `CHANGELOG.md` is hand-maintained in Keep-a-Changelog format.

Local dry-run of a release build: `goreleaser release --snapshot --clean`.
