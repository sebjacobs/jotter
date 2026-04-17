<p align="center">
  <img src="assets/jotter-the-otter-small.png" width="160" alt="Jotter the Otter — a sea otter floating on its back holding a notebook">
</p>

# Jotter

Append-only session log tool for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions. Stores structured JSONL entries in a git-backed data repository, one file per branch.

## Install

```bash
go install github.com/sebjacobs/jotter@latest
```

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

The data directory must be a git repository. Jotter auto-commits every entry and pushes on session finish.

## Shell completion

Jotter offers context-aware completion for the `--project`, `--branch`, and `--type` flags — it reads your actual log store, so tab-completing `--branch` shows only branches that exist for the selected `--project` (sanitised `+` reversed back to `/`).

```bash
jotter completion zsh > /path/to/completions/_jotter    # zsh
jotter completion bash > /etc/bash_completion.d/jotter  # bash
jotter completion fish > ~/.config/fish/completions/jotter.fish
```

For zsh, the completions directory must be on `$fpath` before `compinit` runs. If you manage dotfiles, drop `_jotter` into a tracked `completions/` dir and prepend it to `fpath` in your `.zshrc`.

## Usage

### write

Append a session log entry.

```bash
jotter write --project myapp --branch feature/auth --type start --content "Working on OAuth flow"
jotter write --project myapp --branch feature/auth --type finish --content "OAuth complete" --next "Add refresh token support"
```

Entry types: `start`, `checkpoint`, `note`, `break`, `finish`.

The `--next` flag records what to pick up next session. Finish entries also trigger a git push.

### tail

Show recent entries for a branch.

```bash
jotter tail --project myapp --branch feature/auth              # last entry (default)
jotter tail --project myapp --branch feature/auth --limit 5    # last 5 entries
```

### ls

List projects or branches.

```bash
jotter ls                          # all projects with last activity date
jotter ls --project myapp          # branches in myapp with entry counts
```

### search

Search entries across all logs.

```bash
jotter search "OAuth"                                          # search all logs
jotter search --project myapp --type finish                    # all finish entries in myapp
jotter search --since 2026-04-01                               # entries from April onwards
jotter search "deploy" --project myapp --branch main           # scoped search
```

Filters (`--project`, `--branch`, `--type`, `--since`) can be combined. All filters are AND'd. Search term is case-insensitive and matches against content and next fields.

## Data layout

```
$JOTTER_DATA/
  logs/
    project-a/
      main.jsonl
      feature+auth.jsonl
    project-b/
      main.jsonl
```

Branch names are sanitised: `/` becomes `+` in filenames (e.g. `feature/auth` -> `feature+auth.jsonl`).

Each line is a JSON object:

```json
{"timestamp": "2026-04-15T10:30:00", "type": "start", "content": "Working on OAuth flow"}
{"timestamp": "2026-04-15T11:45:00", "type": "finish", "content": "OAuth complete", "next": "Add refresh token support"}
```

JSON uses Python-compatible spacing (`, ` and `: ` separators) for compatibility with the original Python implementation.

## License

MIT
