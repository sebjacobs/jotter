# Jotter

Append-only, git-backed log for Claude Code sessions. Go rewrite of the Python `session_logger.py`.

## Build & test

The `justfile` is the canonical interface — **always run `just check` before pushing** to catch lint failures CI will otherwise flag.

```bash
just check                   # build + test + lint — mirrors CI, run before every push
just build                   # build binary into bin/
just test                    # run all tests (82 tests)
just lint                    # golangci-lint (same config as CI)
```

Raw commands if needed:

```bash
go build -o bin/jotter .     # build binary (always output to bin/, not repo root)
go test ./...                # run all tests
go test ./cmd/               # command-level integration tests
go test ./internal/          # unit tests for config, entry, storage
go test ./internal/setup/    # unit tests for the setup wizard steps + framework
golangci-lint run            # lint (CI job uses the same config)
```

Tests build the binary once via `TestMain` and run it as a subprocess with a temp git-backed data dir. No mocks.

Local builds get placeholder version info (`dev` / `none` / `unknown`). Release builds via GoReleaser fill in real semver, commit SHA, and build date via `-ldflags`.

## Architecture

```
main.go              -> //go:embed all:skills into skillsFS; cmd.Execute(skillsFS)
skills/              -> embedded session-management SKILL.md files (start, save, stop, break, recover, handover) — installed by `jotter setup`
cmd/
  root.go            -> cobra root command, --version wiring, stores skillsFS
  banner.txt         -> ASCII banner embedded into root command Long description
  version.go         -> version/commit/date vars (ldflags-stamped) + formatter
  write.go           -> append JSONL entry + git commit (local only; push is async); registers the data dir; reconciles branch renames on-branch
  mv.go              -> rename a project's logs dir + git commit the move
  resolve.go         -> resolve --project/--branch from flags or cwd git; onBranch guard for reconciliation
  branch.go          -> `jotter branch` parent: prints current branch (bare)
  branch_mv.go       -> `jotter branch mv` — rename a branch's logs via MoveBranchLogs
  branch_adopt.go    -> `jotter branch adopt` — anchor existing branches for rename tracking
  tail.go            -> read last N entries, render as markdown
  ls.go              -> list projects/branches with metadata
  search.go          -> filter entries by term, type, date, scope
  sync.go            -> fetch + rebase + push; --all walks the registry (skips remoteless repos); syncDataDir + indentWriter for nested per-repo output
  daemon.go          -> `jotter daemon` parent + install/uninstall/status wiring; daemonManager adapts to setup.DaemonManager; renderPlist lives in daemon_darwin.go
  daemon_darwin.go   -> launchd impl: plist render/write, launchctl load/unload, install/uninstall/status, daemonInstalled (build tag: darwin)
  daemon_other.go    -> non-darwin stubs returning "macOS only" (build tag: !darwin)
  config.go          -> print resolved .jotter data_dir for current cwd
  completion.go      -> bash/zsh/fish completion generator
  setup.go           -> `jotter setup` wizard driver; huhPrompter (huh-backed Prompter impl); injects daemonManager
internal/
  config.go          -> resolve data dir by walking up from cwd for .jotter TOML files; falls back to ~/.jotter
  entry.go           -> Entry struct, JSONL marshal (Python-compatible spacing), markdown format
  storage.go         -> path construction, branch sanitisation (/ -> +), glob collection
  branchid.go        -> branch-rename tracking: id anchor + sidecar, AnchorBranch, ReconcileBranch, MoveBranchLogs
  registry.go        -> global registry of data dirs (~/.jotter.d/registry); RegisterDataDir/RegisteredDataDirs; StateDir ($JOTTER_STATE_DIR override) for sync --all + daemon log
  git.go             -> git add/commit/push/fetch/pull-rebase/config + ahead-behind via exec.Command
  color.go           -> TTY-aware ANSI colouring helpers
  setup/
    wizard.go        -> Step interface, State/Status enums, Context, Prompter, DaemonManager, Run driver
    steps.go         -> eight Step implementations + DefaultSteps() (incl. daemonStep)
    settings.go      -> MergePermission for ~/.claude/settings.json
```

## Key conventions

- JSONL uses Python `json.dumps` spacing (`, ` and `: ` separators) for data repo compatibility
- Branch names sanitised: `/` replaced with `+` in filenames
- Branch identity: a stable id lives in the project repo's git config (`branch.<name>.jotter-id`, survives `git branch -m`) and in a `<branch>.jsonl.id` sidecar; lets renames be followed. Sidecars are invisible to the `*.jsonl` globs
- Entry types: `start`, `checkpoint`, `note`, `break`, `stop` (session-end; `finish` is a legacy alias still accepted), `handover` (branch-end distillation written onto `main`)
- Git commit message format: `session: {project}/{branch} {type} {timestamp}`
- Writes commit locally only; pushing is asynchronous — the launchd timer (`jotter daemon`) runs `jotter sync --all` over every registered data repo on an interval. `jotter sync` forces a push now
- Exit code 1 for user-facing errors (missing files, no results, invalid input)

## Release

Prebuilt per-platform binaries are published to GitHub Releases via GoReleaser on `v*` tag push (`.github/workflows/release.yml`). See `CONTRIBUTING.md` for the five-step release cut procedure. `CHANGELOG.md` is hand-maintained in Keep-a-Changelog format.

Local dry-run of a release build: `goreleaser release --snapshot --clean`.
