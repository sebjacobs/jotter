# Jotter

Append-only session log tool for Claude Code sessions. Go rewrite of the Python `session_logger.py`.

## Build & test

```bash
go build -o bin/jotter .     # build binary (always output to bin/, not repo root)
go test ./...                # run all tests (52 tests)
go test ./cmd/               # command-level integration tests
go test ./internal/           # unit tests for config, entry, storage
```

Tests build the binary once via `TestMain` and run it as a subprocess with a temp git-backed data dir. No mocks.

## Architecture

```
main.go              -> cmd.Execute()
cmd/
  root.go            -> cobra root command
  write.go           -> append JSONL entry + git commit (+ push on finish)
  tail.go            -> read last N entries, render as markdown
  ls.go              -> list projects/branches with metadata
  search.go          -> filter entries by term, type, date, scope
internal/
  config.go          -> resolve data dir: JOTTER_DATA env > ~/.config/jotter/config
  entry.go           -> Entry struct, JSONL marshal (Python-compatible spacing), markdown format
  storage.go         -> path construction, branch sanitisation (/ -> +), glob collection
  git.go             -> git add/commit/push via exec.Command
```

## Key conventions

- JSONL uses Python `json.dumps` spacing (`, ` and `: ` separators) for data repo compatibility
- Branch names sanitised: `/` replaced with `+` in filenames
- Entry types: `start`, `checkpoint`, `note`, `break`, `finish`
- Git commit message format: `session: {project}/{branch} {type} {timestamp}`
- `finish` entries trigger git push (non-fatal on failure)
- Exit code 1 for user-facing errors (missing files, no results, invalid input)
