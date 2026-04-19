# Multi-agent support — implementation plan

## Key insight: agentskills.io is a shared spec

Claude Code, Codex CLI, Gemini CLI, and opencode all implement the [agentskills.io specification](https://agentskills.io/specification) — the same `SKILL.md` format with YAML frontmatter (`name`, `description`) and a markdown body. Each client picks its own on-disk skills directory, but the file format is identical.

This collapses the architecture: **one canonical SKILL.md per skill works across every target agent**. No shared-prose extraction, no per-agent frontmatter rendering, no adapter-per-agent Go code that assembles files. The only per-agent concerns that remain are (a) where on disk the skills directory lives and (b) how to wire up command-execution permissions.

## Directory layout

```
integrations/
  skills/                        ← canonical agentskills.io-compliant skills
    start-session/SKILL.md
    save-session/SKILL.md
    finish-session/SKILL.md
    break-session/SKILL.md
    recover-session/SKILL.md     ← Claude-only behaviour (parses ~/.claude/projects/)
                                   but still a valid SKILL.md; non-Claude agents
                                   won't call it
internal/setup/
  agents.go                      ← Agent interface + registry
  claude.go                      ← SkillsDir=~/.claude/skills, wires Bash(jotter:*)
                                   into settings.json
  codex.go                       ← SkillsDir=~/.codex/skills (no permission step)
  gemini.go                      ← SkillsDir=~/.gemini/skills, wires tools.core.allowed
  opencode.go                    ← SkillsDir=~/.config/opencode/skill,
                                   wires permission.bash
```

Each agent file implements a small interface:

```go
type Agent interface {
    Name() string                                // "Claude Code", "Codex", ...
    Detect(home string) bool                     // is this CLI installed?
    SkillsDir(home string) string                // absolute install path
    WirePermission(home string) (changed bool, err error) // no-op for agents without permissions
}
```

The wizard's skills step iterates selected agents, copies `integrations/skills/**` into each `SkillsDir(home)`, then calls `WirePermission`. The existing `installFiles` diff/prompt helper is reused as-is — it only needs a list of `(dest, contents)` pairs.

## Verification before writing adapter code

Before coding Codex / Gemini / opencode adapters, confirm from each client's agentskills.io docs page:
- the exact skills directory path (including whether it's `skills/`, `skill/`, or namespaced under a version)
- whether the client expects subdirectory-per-skill (like Claude) or flat files
- the permission model for `Bash(jotter:*)`-equivalent auto-approval

agentskills.io/clients links to each client's docs. Read them and capture the answers in the spec before writing code — saves per-agent refactors if an assumption turns out wrong.

## Wizard changes

Replace today's single `skillsStep` with:

1. A new **agents** step — runs `Detect(home)` on each registered agent, then a `huh` `MultiSelect` over the detected agents. Stores chosen agent names in `Context.Answers.Agents`.
2. A revised **install** step — for each chosen agent, copies `integrations/skills/**` to its `SkillsDir(home)` via the existing `installFiles` helper, then calls `WirePermission`. Reports counts in the summary.

Existing steps (data dir, remote, .jotter, smoke) stay unchanged. The standalone Claude-detection step goes away — its role is absorbed by the multi-select's detection pass.

## Commit sequence

1. **`goreleaser check` CI + schema comment** — already merged to main.

2. **Refactor `skills/` → `integrations/claude/`** — already committed on this branch (30c0ad5). Mechanical move, no behaviour change.

3. **Flatten `integrations/claude/` → `integrations/skills/`** — the claude/ subdirectory was the old shape; now that every agent consumes the same canonical files, they live at `integrations/skills/`. Update the `//go:embed` path in `main.go` and the default `skillsRoot` in `internal/setup/steps.go`.

4. **Introduce `Agent` interface and Claude adapter** — `internal/setup/agents.go` + `claude.go`. Skills step rewrites to iterate an `[]Agent` (currently length 1). Permission merger moves from `permissionStep` into the Claude adapter. Behaviour-preserving for existing users.

5. **Add multi-select agents step** — replaces `claudeStep`. `huh.MultiSelect` over agents where `Detect` returned true. Skipped entirely on headless runs where the list is empty.

6. **Add Codex adapter** — `internal/setup/codex.go`. Detects `~/.codex/`, installs to its skills dir. No permission wiring.

7. **Add Gemini adapter** — `internal/setup/gemini.go`. Detects `~/.gemini/`, installs skills, merges `tools.core.allowed` in `settings.json`.

8. **Add opencode adapter** — `internal/setup/opencode.go`. Detects `~/.config/opencode/`, installs skills, merges `permission.bash` in `opencode.json`.

9. **Broaden scope in docs and metadata** — README headline, repo description, cobra `Short`/`Long`, GoReleaser `description:`, cask `caveats`. One cross-cutting commit.

10. **Release v0.6.0-rc.1** — exercises cask publish path end-to-end.

11. **Promote to v0.6.0** — after rc.1 installs cleanly on a test machine.

## Testing strategy

- **Unit tests per agent** — assert `SkillsDir(home)` returns the expected path, `Detect` correctly reads the marker dir, `WirePermission` merges the right config snippet.
- **Skills step integration test** — uses a `fstest.MapFS` with a small fixture skills tree and a stub agent list; asserts files land at each agent's `SkillsDir` with correct contents, and that re-running the step is a no-op.
- **E2E smoke on each agent** — manual verification before promoting rc.1 → v0.6.0. Run `/start` and `/finish` in each CLI on a throwaway data dir, confirm entries land in the log.

No CI integration tests per agent — each CLI would need to be installed in the runner, which is more plumbing than the risk warrants. Manual smoke on rc.1 is enough.

## Risks and mitigations

- **Per-agent skills dir paths may be wrong.** Mitigation: read each client's agentskills.io docs page before writing the adapter; capture the answers in this spec.
- **Permission config formats evolve.** Mitigation: keep each adapter's permission merger small and isolated — a future schema change is a one-file patch.
- **`recover-session` is Claude-specific behaviour wrapped in a generic SKILL.md.** It'll load on other agents but its instructions assume `~/.claude/projects/*.jsonl` exists. Mitigation: document in the README that recover is Claude-only; revisit once another agent exposes an accessible transcript format.
- **The multi-select UI is more fragile than single-option confirm prompts.** Mitigation: `huh` ships a well-tested `MultiSelect`; no custom UI code.
