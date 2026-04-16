<p align="center">
  <img src="assets/jotter-the-otter-small.png" width="160" alt="Jotter the Otter — a sea otter floating on its back holding a notebook">
</p>

# Jotter

Append-only session log tool for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions. Stores structured JSONL entries in a git-backed data repository, one file per branch.

## Install

### As a Claude Code plugin (recommended)

```
/plugin marketplace add sebjacobs/jotter
/plugin install jotter@sebjacobs-jotter
```

This installs the session-management skills (`/start`, `/save`, `/break`, `/finish`, `/recover`) and a `/setup-jotter` skill that walks through binary install, data repo setup, and Claude Code permissions.

### Standalone binary

```bash
go install github.com/sebjacobs/jotter@latest
```

## Configuration

Jotter needs to know where your data repository lives. Set the `JOTTER_DATA` environment variable:

```bash
export JOTTER_DATA=~/path/to/session-logs-data
```

Alternatively, write the path to `~/.config/jotter/config`:

```bash
mkdir -p ~/.config/jotter
echo ~/path/to/session-logs-data > ~/.config/jotter/config
```

The data directory must be a git repository. Jotter auto-commits every entry and pushes on session finish.

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
