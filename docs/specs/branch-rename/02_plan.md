# Branch-rename log continuity — implementation plan

Concretises `01_spec.md` into code layout, helper signatures, wiring, and order
of work.

## Code layout

```
internal/
  branchid.go      ← NEW. The anchor + sidecar primitives and the reconcile helper.
  git.go           ← extend: GitConfigGet/Set for branch.<name>.jotter-id.
  storage.go       ← extend: SidecarPath(dataDir, project, branch).
cmd/
  write.go         ← wire reconcile in before the append.
  branch.go        ← gains `mv` and `adopt` subcommands (today it's a leaf command).
```

No new dependencies — `crypto/rand` + `encoding/hex` cover UUID generation,
everything else is the existing `os/exec` git plumbing.

`cmd/branch.go` is today a single leaf command (`jotter branch` prints the current
branch). It becomes a parent with the bare form kept as the default action, plus
`mv` and `adopt` children — mirrors how cobra commands compose elsewhere.

## New git primitives (`internal/git.go`)

```go
// GitConfigGet reads a config value scoped to cwd's repo. Empty string + nil
// error when the key is unset (exit 1 from `git config --get` is not an error
// here).
func GitConfigGet(cwd, key string) (string, error)

// GitConfigSet writes a config value scoped to cwd's repo.
func GitConfigSet(cwd, key, value string) error
```

Key is built as `"branch." + name + ".jotter-id"` — the raw (un-sanitised) branch
name, since git config subsections accept `/`.

## Anchor + sidecar primitives (`internal/branchid.go`)

```go
const anchorKey = "jotter-id"  // -> branch.<name>.jotter-id

func newID() string                                   // 32 hex chars from crypto/rand
func anchorConfigKey(branch string) string            // "branch.<branch>.jotter-id"

// SidecarPath returns logs/<project>/<sanitised-branch>.jsonl.id .
func SidecarPath(dataDir, project, branch string) (string, error)

func ReadSidecar(path string) (string, error)         // "" if absent
func WriteSidecar(path, id string) error              // 0644, single line

// FindLogByID globs logs/<project>/*.jsonl.id and returns the sanitised branch
// basename whose sidecar == id, or "" if none.
func FindLogByID(dataDir, project, id string) (string, error)
```

### The reconcile entry point

```go
// ReconcileBranch ensures the logfile for (project, branch) is named for the
// current branch, following a git rename if one happened. Returns the path the
// caller should append to. Best-effort: any git error is returned as a non-nil
// warn value with a usable fallback path, never as a hard failure.
//
// Runs the spec's algorithm:
//   1. read U = git config branch.<branch>.jotter-id (in cwd's repo)
//   2. no U  -> stamp a fresh id (+ sidecar), adopting <branch>.jsonl if present
//   3. U set -> locate U's logfile; if under a stale name, moveBranchLogs it
func ReconcileBranch(dataDir, cwd, project, branch string) (logPath string, warn error)

// moveBranchLogs git-mv's <old>.jsonl and <old>.jsonl.id to <new> in the data
// repo and commits. Shared by ReconcileBranch and `jotter branch mv`. Refuses
// when <new>.jsonl already exists under a different id (mirrors cmd/mv.go:53).
func moveBranchLogs(dataDir, project, old, new string) error
```

## `write.go` wiring

The reconcile slots in between path construction and the append. It runs **only
on-branch** — guard first, today's behaviour otherwise:

```go
path, err := internal.JSONLPath(dataDir, project, branch)   // existing
// NEW: only reconcile when the write targets the branch cwd is actually on.
if onBranch(cwd, project, branch) {
    if p, warn := internal.ReconcileBranch(dataDir, cwd, project, branch); warn != nil {
        fmt.Fprintf(os.Stderr, "Warning: branch tracking skipped: %v\n", warn)
    } else {
        path = p
    }
}
// ... unchanged: MkdirAll, append, GitCommit ...
```

`onBranch` compares `GitProjectName(cwd) == project && GitCurrentBranch(cwd) ==
branch`, swallowing the not-in-a-repo / detached-HEAD errors as "false". This is
the one guard that keeps scripted and cross-project writes behaving exactly as
today (acceptance criterion 4).

The move commit lands as its own data-repo commit *before* the entry commit, so
the rename and the new entry read as two clean steps in `git log`.

## `branch mv` and `branch adopt` subcommands (`cmd/branch.go`)

```
jotter branch mv <old> <new> [--project P]
```
Resolves `--project` (default: cwd basename), validates both names, calls
`moveBranchLogs`. Thin — same shape as `cmd/mv.go`'s `runMv`. Completion: old arg
from existing branch logfiles, new arg free text.

```
jotter branch adopt [--project P]
```
The migration command (spec §Migration). From cwd's repo: resolve project, list
local branches (`git for-each-ref --format='%(refname:short)' refs/heads`), and
for each branch with a `logs/<project>/<branch>.jsonl` but no anchor, stamp config
+ write sidecar. Idempotent (skip if anchor already set); skip branches whose log
exists but whose git branch is gone. Prints a one-line `adopted N branches`
summary. No data-repo commit needed — sidecars are new files staged by the next
relevant op; adopt commits them in one `chore: adopt branch ids` commit for
cleanliness.

## Testing strategy

Extends the existing subprocess pattern. The wrinkle the current harness doesn't
cover: these tests need cwd inside a **project repo** *and* a known **data dir**
at the same time. Today `runJotterFromGitRepo` (cmd_test.go:1039) gives the
project repo but lets the data dir fall back to `~/.jotter`; `runJotter` gives the
data dir but no project repo.

- **New harness** `runJotterInRepoWithData(t, branch, dataDir, args...)` — a temp
  `git init -b <branch>` project repo as cwd, plus a `.jotter` file in it pointing
  `data_dir` at a temp data repo. One helper unlocks every test below. Reuse the
  worktree-setup shape from `TestProject_ReturnsMainRepoNameFromInsideWorktree`
  (cmd_test.go:1084) for the worktree case.

- **`internal/branchid_test.go`** — unit tests for `newID` (length/uniqueness),
  `SidecarPath`/`ReadSidecar`/`WriteSidecar` round-trip, `FindLogByID`.

- **`cmd/branch_test.go`** — integration, one per acceptance criterion:
  1. fresh-branch write creates `.jsonl` + `.jsonl.id` and stamps config.
  2. `git branch -m old new` in the repo, then write on `new` → entry lands in
     `new.jsonl`, `old.*` gone, data repo has a `rename:` commit.
  3. pre-existing un-anchored logfile adopted on next write, then rename follows.
  4. off-branch / outside-repo write → no config written, no reconcile.
  5. forced git failure (e.g. read-only data dir mid-move) → stderr warning, entry
     still appended.
  6. `branch mv` moves both files + commits; refuses on different-id collision.
  7. `ls`/`search`/`tail` output unchanged by sidecar presence.
  8. `branch adopt` stamps all existing local branches, idempotent on re-run,
     skips deleted branches.

  Plus a worktree variant of (2): rename from inside a `gwt`-style worktree still
  reconciles, proving the shared-config assumption.

## Order of work

Each step leaves the tree green (`just check`) and is one atomic commit.

1. **`internal/git.go` config primitives** — `GitConfigGet`/`Set` + unit coverage
   via the new harness. Pure plumbing. One commit.
2. **`internal/branchid.go` primitives** — id, sidecar, `FindLogByID`, with
   `branchid_test.go`. No `write.go` wiring yet. One commit.
3. **`moveBranchLogs` + `ReconcileBranch`** — the reconcile core, unit-tested
   directly (criteria 2, 3, 6-collision). One commit.
4. **Wire `ReconcileBranch` into `write.go`** behind the `onBranch` guard —
   integration criteria 1, 2, 3, 4, 5. One commit.
5. **`jotter branch mv`** — subcommand over `moveBranchLogs`, criterion 6. One
   commit.
6. **`jotter branch adopt`** — migration command, criterion 8. One commit.
7. **Docs** — README + CLAUDE.md: branch-tracking behaviour, the project-repo
   `branch.<name>.jotter-id` side-effect, `branch mv`/`adopt` usage; ROADMAP entry
   moved to Shipped. One commit.
8. **Cut a release** — per `CONTRIBUTING.md`: CHANGELOG, tag, push.

Steps 1-3 are pure internals with no behaviour change shipped; step 4 is where the
auto-follow goes live. That ordering keeps every intermediate commit green and
independently reviewable.

## Companion: the `proj` cross-repo sweep (not in this repo)

Out of scope here (spec §Out of scope), tracked so it isn't lost: once `jotter
branch adopt` ships, add a `proj`-side loop in the dotfiles repo that runs it in
each known checkout — `proj` owns the project inventory jotter lacks. Same
composition as the existing `proj mv` → `jotter mv` call.

## Out of this plan

- The rest of `jotter project` lifecycle (`rm`, `info`, `path`) — separate spec.
- The `.jotter` `project_name` lever — project-lifecycle spec.
- git-hook eager detection — unnecessary given the lazy anchor.
- The cross-repo `proj` sweep — dotfiles, not jotter.
