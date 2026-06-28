<p align="center">
  <img src="assets/jotter-the-otter-small.png" width="160" alt="Jotter the Otter — a sea otter floating on its back holding a notebook">
</p>

# Jotter

> Jot it, keep it, find it later.

An append-only, git-backed log for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions. Stores structured JSONL entries in a git-backed data repository, one file per branch.

## Why jotter

A Claude Code session generates context worth keeping — the goal you set, decisions made along the way, blockers, what to pick up next time. But that context doesn't belong in your project's git history (it's not about the code), and it doesn't survive `/clear` or the end of the day.

Jotter gives each session a durable log of its own: structured checkpoints committed to a separate git repo, searchable across every project and branch you've worked on. Start a session knowing where you left off. Finish one knowing it's written down.

## How it works

- **In Claude Code**, you type `/start`, `/save`, `/break`, `/stop`, or `/recover`. The skills (installed by `jotter setup`) handle the rest — they know what to capture and call `jotter write` for you.
- **Entries land as JSONL** in a separate git repo, one file per project/branch. One commit per entry. Pushing to the remote happens asynchronously — a background timer (`jotter daemon`) pushes every logged repo on an interval, so writes never block on the network.
- **To look back**, use `jotter tail` to replay a branch, `jotter ls` to browse what's been logged, or `jotter search` to grep across everything — by project, branch, type, or date.

## Install

### Homebrew (macOS and Linux)

```bash
brew install sebjacobs/tap/jotter
```

Upgrades come with `brew upgrade jotter`. The tap lives at [github.com/sebjacobs/homebrew-tap](https://github.com/sebjacobs/homebrew-tap) and the cask is updated automatically on every release.

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/sebjacobs/jotter/main/install.sh | sh
```

Downloads the latest prebuilt binary for your platform (macOS arm64/amd64, Linux arm64/amd64), verifies its SHA-256 checksum, and installs it to `$HOME/.local/bin`. Override the target directory with `JOTTER_INSTALL_DIR=/path/to/bin`, or pin a version with `JOTTER_VERSION=v0.1.0`.

If `$HOME/.local/bin` isn't already on your `PATH`, the script prints the `export` line you need.

### `go install`

If you have a Go toolchain and prefer source builds (or you're on an unsupported platform like Windows):

```bash
go install github.com/sebjacobs/jotter@latest
```

Binaries built this way report `jotter dev` for `--version`; release tags only land in binaries from the prebuilt flow above.

## Setup

Once `jotter` is on your PATH, the fastest way to wire it into Claude Code is the interactive setup wizard:

```bash
jotter setup
```

It takes you through seven steps in one go — detects Claude Code, prompts for a data directory (default `~/session-logs-data`), initialises it as a git repo, optionally wires a git remote, writes your `~/.jotter` config, installs the session-management skills (`/start`, `/save`, `/stop`, `/break`, `/recover`, `/handover`), grants the `Bash(jotter:*)` permission in `~/.claude/settings.json`, and runs a write-and-read-back smoke test. Re-running is idempotent — it detects existing state and only updates what's changed.

If you'd rather wire things up by hand, the `Configuration` section below describes the same artefacts the wizard produces.

## Using jotter from Claude Code

The primary interface is the session-management skills — you don't call `jotter write` by hand during a session.

| Command     | When you use it |
|-------------|-----------------|
| `/start`    | Beginning a session. Reads recent logs, proposes a goal, writes a `start` entry. |
| `/save`     | Mid-session checkpoint. Jot down a decision, a finding, or progress. |
| `/break`    | Stepping away. Snapshots state so you can pick up cleanly. |
| `/stop`     | Wrapping up a session (formerly `/finish`, still accepted). Writes a summary and records what's next (the push happens in the background). |
| `/handover` | A feature branch is done for good. Distils its log onto `main` so the context survives deleting the branch. |
| `/recover`  | Picking up a crashed or unfinished session. Reconstructs context from the last entries. |

A typical day:

```text
/start          # "continuing the OAuth work — goal: ship refresh tokens"
...work happens, Claude writes code, you review...
/save           # "refresh-token flow implemented, tests passing"
...more work, hit a blocker...
/save           # "blocked on the cookie-vs-header decision — see note"
/stop           # writes summary, records 'cookie vs header: decide next session'
```

Next session, `/start` reads that `stop` entry and reminds you where you were.

## CLI reference

When you want to query past sessions or write entries outside of a Claude Code session, the CLI has these subcommands.

### write

Append a session log entry.

```bash
jotter write --project myapp --branch feature/auth --type start --content "Working on OAuth flow"
jotter write --project myapp --branch feature/auth --type stop --content "OAuth complete" --next "Add refresh token support"
```

Entry types: `start`, `checkpoint`, `note`, `break`, `stop`, `handover`. (`finish` is a legacy alias for `stop`, still accepted.)

The `--next` flag records what to pick up next session. Writes only commit locally — the remote is updated asynchronously by the background timer (see `daemon` below), or on demand with `jotter sync`.

### mv

Rename a project's logs.

```bash
jotter mv old-name new-name
```

A project's jotter name is the basename of its git toplevel, so renaming the project directory orphans its logs under the old name. `mv` renames `logs/<old-name>` to `logs/<new-name>` and commits the move locally (it refuses to overwrite an existing destination).

### branch

Print the current branch, and manage per-branch logs.

```bash
jotter branch                      # print the current git branch
jotter branch mv old new           # rename a branch's logs (logs/<project>/old.jsonl -> new.jsonl)
jotter branch adopt                # anchor existing branches so future renames are tracked
```

Logs are stored one file per branch, so a `git branch -m` would otherwise orphan a branch's history under the old name. Jotter handles this automatically: on the first write to a branch it records a stable id in your project repo's git config (`branch.<name>.jotter-id`) — which survives the rename, because git moves the whole `[branch]` config section — and a matching `<branch>.jsonl.id` sidecar next to the logfile. The next write after a rename spots the mismatch and moves the logfile into place, so history stays continuous with no action from you.

`branch mv` is the manual equivalent for when you won't write to a branch again (e.g. a merged feature branch). `branch adopt` migrates a repo whose branches predate this feature — run it once so a branch renamed *before* its next write is still followed. Both default `--project` to the git toplevel basename.

### tail

Show recent entries for a branch.

```bash
jotter tail --project myapp --branch feature/auth              # last entry (default)
jotter tail --project myapp --branch feature/auth --limit 5    # last 5 entries
```

### ls

List projects or branches.

```bash
jotter ls                                              # all projects with last activity date
jotter ls --project myapp                              # branches in myapp with entry counts
jotter ls --project myapp --branch feature/auth        # entries on that branch, one per line (timestamp, type, title)
```

A typical `ls --project --branch` run looks like:

```
2026-04-11 12:00  stop        OAuth flow — initial implementation done, tests green.
2026-04-14 21:49  checkpoint  Refresh-token spike complete — committed. Rotation handling next.
2026-04-16 15:45  stop        PR merged — OAuth end-to-end shipped.
```

Titles are extracted from the first non-empty line of each entry's content, with basic markdown markers stripped.

### search

Search entries across all logs.

```bash
jotter search "OAuth"                                          # search all logs
jotter search --project myapp --type stop                      # all stop entries in myapp
jotter search --since 2026-04-01                               # entries from April onwards
jotter search "deploy" --project myapp --branch main           # scoped search
```

Filters (`--project`, `--branch`, `--type`, `--since`) can be combined. All filters are AND'd. Search term is case-insensitive and matches against content and next fields.

### sync

Push pending entries to the remote.

```bash
jotter sync            # fetch, rebase local entries, push — for the current cwd's data repo
jotter sync --all      # do the same for every registered data repo that has a remote
```

Writes only commit locally, so `sync` is how committed entries reach the remote. It fetches, rebases your local entries on top of anything the remote gained, then pushes — also the recovery path when a previous push failed (offline, or the remote moved on). `--all` walks the registry of every data repo you've written to and is what the background timer runs; data repos without a remote are skipped.

### daemon

Manage the background push timer (macOS/launchd).

```bash
jotter daemon install                  # install the timer (pushes every 5 minutes)
jotter daemon install --interval 60    # ...or choose the interval, in seconds
jotter daemon status                   # is it installed and loaded? plus a recent-log tail
jotter daemon uninstall                # remove it
```

`install` writes a launchd LaunchAgent that runs `jotter sync --all` on an interval, so every data repo you log to is pushed in the background and writes never block on the network. `jotter setup` offers to install it for you once a remote is configured. On non-macOS platforms, schedule `jotter sync --all` with cron or a systemd timer instead.

The agent's label defaults to `com.jotter`. If you manage your launchd agents under a single reverse-DNS prefix (e.g. a service manager that globs `<prefix>.*`), export `LAUNCHD_PREFIX` and jotter names its agent `<prefix>.jotter` so it joins that set — `LAUNCHD_PREFIX=com.example` yields `com.example.jotter`. `jotter daemon status` prints the resolved label. The variable is purely opt-in; unset, jotter uses `com.jotter` and nothing else is required.

### project

Print the current project name.

```bash
jotter project        # basename of the git toplevel for the current directory
```

A project's jotter name is the basename of its git toplevel — the same value the other commands default `--project` to. `project` exposes it directly so skill templates and scripts can pass `--project "$(jotter project)"` without reimplementing the git plumbing. It errors if the current directory isn't inside a git repo.

## Configuration

Jotter is configured via a `.jotter` TOML file. Drop one in your home directory for a global default, and optionally one at the root of any project that should use a different data dir:

```toml
# ~/.jotter
data_dir = "~/session-logs-data"
```

```toml
# ~/Projects/private-repo/.jotter  (overrides ~/.jotter for anything inside this directory)
data_dir = "~/session-logs-private"
```

When jotter runs, it walks up from the current directory looking for a `.jotter` file. The first one found wins; if nothing is found on the walk, it falls back to `~/.jotter`. One rule, no env vars, no XDG config dir.

Supported keys:

- `data_dir` (required) — path to the session-logs data directory. Leading `~` expands to the user's home dir. Relative paths resolve against the directory containing the `.jotter` file.

Run `jotter config` to see which `.jotter` file jotter would use from your current cwd and the resolved `data_dir`. Use this before `jotter write` if you're unsure which store an entry will land in.

The data directory must be a git repository. Jotter auto-commits every entry locally; pushing to the remote happens asynchronously via the background timer (`jotter daemon`) or on demand with `jotter sync`.

## Shell completion

Jotter offers context-aware completion for the `--project`, `--branch`, and `--type` flags — it reads your actual log store, so tab-completing `--branch` shows only branches that exist for the selected `--project` (sanitised `+` reversed back to `/`).

```bash
jotter completion zsh > /path/to/completions/_jotter    # zsh
jotter completion bash > /etc/bash_completion.d/jotter  # bash
jotter completion fish > ~/.config/fish/completions/jotter.fish
```

For zsh, the completions directory must be on `$fpath` before `compinit` runs. If you manage dotfiles, drop `_jotter` into a tracked `completions/` dir and prepend it to `fpath` in your `.zshrc`.

## Data layout

```
$JOTTER_DATA/
  logs/
    project-a/
      main.jsonl
      main.jsonl.id
      feature+auth.jsonl
      feature+auth.jsonl.id
    project-b/
      main.jsonl
      main.jsonl.id
```

Branch names are sanitised: `/` becomes `+` in filenames (e.g. `feature/auth` -> `feature+auth.jsonl`).

Each logfile has a `.jsonl.id` sidecar holding the branch's stable id (see [`branch`](#branch)). It's how a renamed branch's logs are matched back to their history; `ls`, `tail`, and `search` ignore it.

Each line is a JSON object:

```json
{"timestamp": "2026-04-15T10:30:00", "type": "start", "content": "Working on OAuth flow"}
{"timestamp": "2026-04-15T11:45:00", "type": "stop", "content": "OAuth complete", "next": "Add refresh token support"}
```

JSON uses Python-compatible spacing (`, ` and `: ` separators) for compatibility with the original Python implementation.

## Development

Clone the repo and build from source:

```bash
git clone https://github.com/sebjacobs/jotter.git
cd jotter
go build -o bin/jotter .     # binary goes to bin/, not the repo root
go test ./...                # run the full suite
```

A [`justfile`](justfile) wraps the common dev tasks. Install [`just`](https://github.com/casey/just) (`brew install just`) and run:

```bash
just            # list recipes
just build      # build binary into bin/
just test       # run all tests
just lint       # run golangci-lint (same config as CI)
just check      # run build + test + lint — do this before pushing
```

`just check` mirrors what CI runs, so a green local check means a green CI run.

Architecture breakdown — what lives where (skills, commands, internal packages) — is documented in [`CLAUDE.md`](CLAUDE.md). Release process (cutting a tagged release, bumping the changelog) is in [`CONTRIBUTING.md`](CONTRIBUTING.md).

Local builds report `jotter dev` for `--version`; real version info (semver tag, commit SHA, build date) is stamped in via `-ldflags` only on GoReleaser builds from tag pushes.

## License

MIT
