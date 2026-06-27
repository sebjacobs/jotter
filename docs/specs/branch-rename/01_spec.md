# Branch-rename log continuity — v1 spec

## Why

A project's logs live at `logs/<project>/<branch>.jsonl` — the branch is the
*filename*, not a field inside any entry (`internal/storage.go:57`,
`internal/entry.go:14`). That means renaming a git branch with
`git branch -m old new` silently orphans its history: the file stays
`logs/<project>/old.jsonl`, and the next `jotter write` on `new` starts a fresh
`logs/<project>/new.jsonl`. The session narrative splits across two files with no
link between them.

This is the exact sibling, one level down, of the project-rename problem that
`jotter mv` already solves for directories (`cmd/mv.go`). The roadmap already
names the eager form of the fix under *Next → `jotter project` lifecycle
subcommands*: "Symmetric `jotter branch mv <old> <new>` for the branch-rename
case."

The design question this spec settles is not *whether* to offer that manual
command, but whether a rename can be made to **follow automatically** without the
user remembering to run anything — and how, given branch names are mutable and
git provides no stable per-branch identity.

## Approach — lazy reconciliation anchored on a stamped branch id

The chosen mechanism (decided 2026-06-27) is a **git-config anchor**, reconciled
lazily on the next write. It rests on one durable git fact:

> `git branch -m old new` moves the entire `[branch "old"]` config section to
> `[branch "new"]`. Any key stored under that section — including a custom one —
> travels with the branch across the rename.

This is the same mechanism that lets upstream-tracking config survive a rename.
We exploit it to give each branch a stable identity that outlives its name.

### The anchor

On the first `jotter write` against a branch, jotter stamps a UUID into the
*project* repo's config:

```
git config branch.<name>.jotter-id <uuid>
```

After a rename, `git config branch.<new>.jotter-id` returns the *same* UUID it
had under the old name — that is the signal that `new` is not a new branch but a
renamed one.

### The back-reference

The config tells us "branch `new` has identity U". To act on a rename we also
need to find *which logfile* is U's — possibly still under its old name. Each
logfile carries a sidecar recording its identity:

```
logs/<project>/<branch>.jsonl       # the log (unchanged)
logs/<project>/<branch>.jsonl.id    # one line: the UUID
```

A per-file sidecar is preferred over a single central `index.json` because it is
git-merge-friendly (no shared mutable file to conflict on in the data repo),
moves atomically alongside its `.jsonl` on rename, and is invisible to every
existing glob — `CollectPaths` matches `*.jsonl`, which does not match
`*.jsonl.id` (`internal/storage.go:85,91`). The central-index alternative is
noted under Open questions.

### Reconcile algorithm (on `jotter write`)

Run only when the write targets the branch the caller is actually on — i.e. the
`--project`/`--branch` args resolve to the same project and branch as cwd's repo
(`GitProjectName(cwd)` and `GitCurrentBranch(cwd)`). For a cross-project or
off-branch scripted write, or outside a git repo, or detached HEAD: **skip
reconciliation entirely** and behave exactly as today. The magic is scoped to the
normal session flow; scripted writes stay predictable.

When in scope:

1. Read `U = git config branch.<branch>.jotter-id` from cwd's repo.
2. **No `U`** — branch not yet anchored. Generate a UUID, write it to config and
   to `<branch>.jsonl.id`, then append. (If `<branch>.jsonl` already exists from
   before this feature, this is the lazy adoption path — it gets anchored on its
   next write, no migration step needed.)
3. **`U` present** — find the logfile whose sidecar equals `U` (fast path: check
   `<branch>.jsonl.id`; otherwise glob `logs/<project>/*.jsonl.id`):
   - sidecar found at `<branch>.jsonl.id` → already current, just append.
   - sidecar found at `<old>.jsonl.id` (basename ≠ current) → **rename detected**:
     `git mv` both `<old>.jsonl` and `<old>.jsonl.id` to `<new>` in the data repo,
     commit the move (`rename: logs/<project>/<old> -> <new>`), then append.
   - no sidecar carries `U` (config anchored but file gone) → treat as new:
     create `<branch>.jsonl`, write its sidecar `U`, append.

**Reconciliation is never allowed to fail a write.** Any git error in steps 1–3
is warned to stderr and the write falls through to today's behaviour (append to
`<branch>.jsonl`). The log entry is sacred; auto-follow is best-effort, matching
the existing "push failure is non-fatal" stance (`cmd/write.go:95`).

## Eager companion — `jotter branch mv`

Lazy follow-on-write doesn't cover a branch you rename and then never write to
again (a merged feature branch, say), and there's value in an explicit recovery
command. So this spec also lands the manual form named in the roadmap:

```
jotter branch mv <old> <new> [--project P]
```

It performs the same data-repo move as the reconciler (`git mv` of `.jsonl` +
`.jsonl.id`, commit), defaulting `--project` to the cwd-resolved name. Both paths
call one shared internal helper — build that helper first, then the reconciler
and the command are thin callers of it. Mirrors `cmd/mv.go`'s existing shape and
its refuse-to-overwrite guard.

## In scope (v1)

- Stamp `branch.<name>.jotter-id` in the project repo on first write to a branch.
- Per-logfile `<branch>.jsonl.id` sidecar recording the UUID.
- Lazy rename reconciliation on `jotter write`, scoped to the on-branch case,
  non-fatal on any git error.
- Lazy adoption of pre-existing (pre-feature) logfiles on their next write.
- **Proactive migration** of existing repos (see below) — `jotter branch adopt`.
- `jotter branch mv <old> <new> [--project P]` as the manual counterpart.
- Shared internal `moveBranchLogs(dataDir, project, old, new)` helper backing both.

## Migration of existing repos

Lazy adoption alone leaves a hole: an existing branch renamed *before* its next
write never gets anchored, so that rename can't be followed — the exact bug, for
the pre-feature population. Anchors must therefore be stampable **proactively**,
not only on next write.

The anchor lives in each *project* repo's config, and jotter only knows project
*names* (data-repo dir basenames), not where the repos live on disk. So migration
is inherently **per-repo** — it has to run from inside each one:

```
jotter branch adopt [--project P]   # run inside a project repo
```

From cwd's repo it resolves the project, lists local branches, and for every
branch that has a `logs/<project>/<branch>.jsonl` but no anchor yet, stamps
`branch.<name>.jotter-id` + writes the sidecar. Idempotent — re-running skips
already-anchored branches. Branches whose logs exist but whose git branch is gone
(merged/deleted) are skipped: they can't be renamed, so they need no anchor (the
sidecar alone, if wanted, is a follow-up, not v1).

**Cross-repo driver lives outside jotter.** Sweeping every repo can't be jotter's
job — it has no repo inventory. `proj`, which *does* know every project's path, is
the natural orchestrator: a `proj`-side loop that runs `jotter branch adopt` in
each checkout. That belongs in the dotfiles `proj` helper, not here; this spec
only commits to jotter exposing the single-repo `adopt` command it would call.
(This is the same CLI-composition pattern as `proj mv` → `jotter mv`.)

## Out of scope

- **The rest of the `jotter project` lifecycle suite** (`rm`, `info`, `path`) —
  separate spec; this one is only branch-rename continuity.
- **The `.jotter` `project_name` config lever** — orthogonal; lives with the
  project-lifecycle spec.
- **Eager detection via git hooks** (the `reference-transaction` option) — the
  lazy anchor makes a hook unnecessary for the common case; revisit only if
  follow-on-write latency proves insufficient.
- **The cross-repo migration sweep** — driving `adopt` across many repos is a
  `proj` concern (it owns the project inventory); jotter ships only the
  single-repo `adopt` command.
- **The "entries land on the starting branch" roadmap item** — related session
  fragmentation, but a distinct fix (skill templates, not storage).

## Edge cases

- **Sanitisation.** Filenames sanitise `/`→`+` (`SanitiseBranch`); the git config
  subsection uses the *real* branch name (`branch.feature/foo.jotter-id` is valid
  — git supports `/` in subsection names). Reconciliation compares sanitised
  filenames; the config key uses the raw name.
- **Worktrees.** Branch config lives in the shared `.git/config` (the common dir),
  not per-worktree, and `GitProjectName` already resolves to the main repo's
  basename (`internal/git.go:12`). So stamping and reconciling work uniformly
  whether the write happens in the main checkout or a `gwt` worktree.
- **Delete + recreate same name.** `git branch -d` removes the `[branch "x"]`
  section, so a recreated branch has no anchor and adopts the existing
  `<name>.jsonl` on next write — same name, same file. Acceptable.
- **Rename collision.** Renaming `old`→`new` when `<new>.jsonl` already exists
  under a *different* UUID: refuse the move, warn, and append to the current-name
  file rather than merge two distinct histories (mirrors `cmd/mv.go:53`'s
  refuse-to-overwrite). Flagged under Open questions.
- **`--branch` explicitly mismatching cwd.** Skipped (reconciliation only runs
  on-branch), so scripted/cross-project writes are untouched.

## Side-effect disclosure

This feature introduces a new kind of side-effect: jotter writes into the
**user's project repo** config (`branch.<name>.jotter-id`), where today it only
reads git state from cwd and writes solely to its own data repo. The key is inert
(git ignores unknown branch keys), travels with the branch, and is harmless to
delete. No opt-out (decided 2026-06-27) — the key is inert and branch tracking is
the whole point. It must be documented in the README/CLAUDE.md.

## Acceptance criteria

1. `jotter write` on a fresh branch creates `<branch>.jsonl` + `<branch>.jsonl.id`
   and `git config branch.<branch>.jotter-id` returns a UUID.
2. `git branch -m old new` then `jotter write` on `new` → the next write lands in
   `logs/<project>/new.jsonl`, `old.jsonl`/`old.jsonl.id` no longer exist, and the
   data repo has a `rename: …` commit. History is continuous.
3. A logfile created before the feature (no sidecar, no config anchor) is adopted
   on its next write — gets a sidecar + config anchor — and a subsequent rename
   then follows as in (2).
4. A write whose `--branch` ≠ cwd's current branch (or run outside a git repo)
   behaves exactly as today — no config written, no reconciliation.
5. A simulated git failure during reconciliation warns on stderr but the entry is
   still appended to a valid logfile (write never fails).
6. `jotter branch mv old new` moves `.jsonl` + `.jsonl.id` and commits; refuses
   when `new.jsonl` already exists under a different id.
7. `ls` / `search` / `tail` ignore `*.jsonl.id` sidecars (no new noise in output).
8. `jotter branch adopt` run inside a repo with existing un-anchored branch logs
   stamps an anchor + sidecar for every still-existing local branch, is idempotent
   on re-run, and skips branches whose git branch no longer exists.

## Open questions

- **Sidecar vs central index.** Sidecar chosen for git-friendliness; revisit if
  the per-file proliferation feels noisy in the data-repo tree. A single
  `logs/<project>/.jotter-branches.json` is the alternative.
- **Collision policy.** Refuse-and-warn (above) is the safe default. Is there ever
  a case where merging two histories into the surviving file is wanted? Probably
  not — keep them separate.

## Next stage

- `02_plan.md` — concrete file layout (`internal/branchid.go`?), the shared
  `moveBranchLogs` + anchor-stamp helpers, `write.go` reconcile wiring,
  `cmd/branch.go` gaining `mv` and `adopt` subcommands, test plan (subprocess
  integration tests with a temp project repo + temp data repo), and order of work.
