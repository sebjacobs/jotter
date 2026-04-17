# Setup wizard — implementation plan

Concretises `01_spec.md` into code layout, dependencies, and order of work.

## Code layout

```
main.go                         ← //go:embed all:skills — passes embed.FS into cmd.Execute
cmd/
  setup.go                      ← cobra command; parses flags, calls internal/setup
internal/
  setup/
    wizard.go                   ← Step interface, step runner, summary table
    detect.go                   ← shared filesystem / config detection helpers
    settings.go                 ← settings.json merge (separately tested)
    steps/
      claude.go                 ← step 1: Claude Code presence
      datadir.go                ← step 2: data dir prompt + git init
      remote.go                 ← step 3: remote URL + first push
      config.go                 ← step 4: .jotter write
      skills.go                 ← step 5: copy embedded skills
      permission.go             ← step 6: settings.json merge
      smoke.go                  ← step 7: write + tail smoke test
skills/                         ← already ported (five SKILL.md files)
```

`cmd/setup.go` is thin — flag parsing + dispatch. All real logic is in `internal/setup/` so it's unit-testable without running the binary.

## Why `main.go` embeds, not `cmd/`

Go's `//go:embed` can't use `..` patterns. The directive must be at or above the embedded files. Three options considered:

1. **Embed from `main.go`** (chosen) — skills stay at repo root where dotfiles users expect them; one extra parameter threaded through `Execute`.
2. Move `skills/` into `cmd/setup/skills/` — co-located but hides them inside an implementation-detail dir; less discoverable.
3. Move `skills/` into `internal/setup/skills/` — same discoverability problem, buried one level deeper.

Option 1 wins on discoverability with a trivial plumbing cost.

## Dependencies

Add one:

- `github.com/charmbracelet/huh` — interactive prompt forms for the wizard

Everything else is stdlib:

- `embed` — skill bundling
- `os/exec` — `git` invocations for data repo init / first push
- `encoding/json` — `settings.json` merge
- Existing internal packages (`internal/config`, `internal/git`, `internal/storage`) reused for data-dir resolution and smoke test

## Step model

```go
type Step interface {
    Name() string
    Detect(ctx *Context) (State, error)   // already-done / needs-prompt / not-applicable
    Run(ctx *Context) (Result, error)     // execute the step
}

type Result struct {
    Status  string  // "ok" | "skipped" | "updated" | "failed"
    Message string  // one-line summary for the end-of-run table
}

type Context struct {
    Home      string       // user home, injectable for tests (t.Setenv HOME)
    SkillsFS  embed.FS     // threaded from main
    Answers   *Answers     // accumulated user input (data dir, remote, etc)
    Prompter  Prompter     // interface wrapping huh so tests can inject canned answers
}
```

The runner iterates steps, prints ✓ / ↷ / ✎ / ✗ per step, and renders a final summary table. Failures short-circuit; idempotent re-runs pick up from the failure point.

## Testing strategy

Matches the existing `internal/` unit + `cmd/` integration split:

- **Per-step unit tests** — each step's `Detect` and `Run` tested in isolation with `t.TempDir()` + `t.Setenv("HOME", …)`. No network, no real `~/.claude`.
- **`internal/setup/settings.go`** — exhaustive table tests for the `settings.json` merge (empty file, existing permissions, duplicate entry, malformed JSON, nested structure).
- **`cmd/setup_test.go`** — end-to-end invocation with a `--non-interactive` flag and preset answers against a temp HOME with a fake `~/.claude/` tree.
- **No tests for `huh` rendering** — form UI isn't ours to test. `Run` functions take a `Prompter` interface, so tests inject canned answers.

## Order of work

Each step leaves the tree green and commits in a single atomic chunk. Steps 1–2 are bootstrap wiring; 3 is the framework; 4–10 flesh out the actual steps.

1. **Wire `//go:embed all:skills`** — add embed directive in `main.go`, thread `embed.FS` through `cmd.Execute`. Add placeholder `cmd/setup.go` that prints `embedded N skills` to prove the wiring. One commit.
2. **Add `github.com/charmbracelet/huh` dependency** — `go get` and confirm it compiles. Separate commit for a clean diff.
3. **Step framework** — `internal/setup/wizard.go` with `Step` interface and runner, `Prompter` abstraction, summary-table renderer. No real steps yet — one stub step wired up and exercised by a test. One commit.
4. **`internal/setup/settings.go`** — settings.json merge logic in isolation, with table tests. Pure function; can be built before the step that uses it. One commit.
5. **Implement steps 4 (.jotter write) and 6 (settings.json merge)** — simplest steps, both pure file manipulation. Each with tests. Two commits.
6. **Implement step 2 (data dir + git init) and step 3 (remote + first push)** — involve `exec.Command("git", …)`. Mock by injecting a git runner interface. Two commits.
7. **Implement step 5 (skills install)** — walks the embedded FS, writes files to `~/.claude/skills/`, detects and warns on symlinks. One commit.
8. **Implement step 1 (Claude Code detect) and step 7 (smoke test)** — steps 1 is trivial; 7 invokes existing `jotter write` and `jotter tail` paths internally. Two commits.
9. **Wire `huh` prompts** — swap the stub `Prompter` for a real `huh`-backed implementation. Test the wizard end-to-end against the real prompt library on a scratch HOME. One commit.
10. **README + CLAUDE.md update** — add `jotter setup` to the README install section (runs after `install.sh`), update CLAUDE.md architecture map with `cmd/setup.go` and `internal/setup/`. One commit.
11. **Cut v0.2.0** — per `CONTRIBUTING.md`: CHANGELOG, commit, tag, push.

## Out of this plan

- Auto-upgrade, uninstall flows
- Multi-agent generalisation (Codex, Aider, Cursor)
- Custom skill sets (user-overrideable skill directory)
- Windows support

All deferred to future specs if/when they become real needs.
