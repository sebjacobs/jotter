// Package setup implements the interactive 'jotter setup' wizard.
//
// The wizard is a linear sequence of Steps. Each step:
//   - Detects current state (already done, needs prompt, not applicable)
//   - Runs if needed, prompting the user via a Prompter
//   - Reports a Result with a status and a one-line message
//
// The runner prints a symbol per step as it executes, then a summary table at
// the end. Failures short-circuit — the wizard is idempotent, so re-running
// picks up where it broke.
package setup

import (
	"embed"
	"fmt"
	"io"
)

// State describes what Detect found before a step runs.
type State int

const (
	// NeedsRun means the step has work to do.
	NeedsRun State = iota
	// AlreadyDone means the step's target state already exists; skip.
	AlreadyDone
	// NotApplicable means this step shouldn't run in this environment
	// (e.g. Claude Code isn't installed).
	NotApplicable
)

// Status is the final outcome reported in the summary table.
type Status string

const (
	StatusOK      Status = "ok"
	StatusSkipped Status = "skipped"
	StatusUpdated Status = "updated"
	StatusFailed  Status = "failed"
)

// Result is what a Step.Run returns.
type Result struct {
	Status  Status
	Message string
}

// Step is one unit of work in the wizard.
type Step interface {
	Name() string
	Detect(ctx *Context) (State, error)
	Run(ctx *Context) (Result, error)
}

// Context is passed to every step. It bundles the injected dependencies
// (home dir, embedded skills, prompter) plus mutable state accumulated as
// the wizard progresses (user answers).
type Context struct {
	Home       string   // user home dir — injectable for tests via t.Setenv
	SkillsFS   embed.FS // embedded skills tree
	SkillsRoot string   // root inside SkillsFS (default "skills"; tests may override)
	Prompter   Prompter // prompt abstraction; tests inject canned answers
	Answers    *Answers // accumulated user input across steps
	Out        io.Writer
}

// skillsRoot returns the embed-tree root path for skills, defaulting to
// "skills" when Context.SkillsRoot is unset.
func (c *Context) skillsRoot() string {
	if c.SkillsRoot == "" {
		return "skills"
	}
	return c.SkillsRoot
}

// Answers accumulates user input during the wizard so later steps can read
// decisions from earlier steps (e.g. step 4 .jotter write needs the data_dir
// chosen in step 2).
type Answers struct {
	DataDir   string
	RemoteURL string // optional; empty = user skipped
}

// Prompter is the interface steps use to ask the user questions. Production
// uses a huh-backed implementation; tests inject a stub.
type Prompter interface {
	// Confirm prompts a yes/no question with a default answer.
	Confirm(question string, defaultYes bool) (bool, error)
	// Input prompts for a freeform text answer with a default.
	Input(question, defaultValue string) (string, error)
}

// Run executes the given steps in order, printing per-step symbols as it goes
// and a summary table at the end. First failure short-circuits.
func Run(ctx *Context, steps []Step) error {
	results := make([]stepOutcome, 0, len(steps))
	for _, s := range steps {
		state, err := s.Detect(ctx)
		if err != nil {
			return fmt.Errorf("detect %q: %w", s.Name(), err)
		}
		switch state {
		case AlreadyDone:
			results = append(results, stepOutcome{name: s.Name(), result: Result{Status: StatusSkipped, Message: "already done"}})
			fmt.Fprintf(ctx.Out, "  ↷ %s — already done\n", s.Name())
			continue
		case NotApplicable:
			results = append(results, stepOutcome{name: s.Name(), result: Result{Status: StatusSkipped, Message: "not applicable"}})
			fmt.Fprintf(ctx.Out, "  ↷ %s — not applicable\n", s.Name())
			continue
		}

		r, err := s.Run(ctx)
		if err != nil {
			results = append(results, stepOutcome{name: s.Name(), result: Result{Status: StatusFailed, Message: err.Error()}})
			fmt.Fprintf(ctx.Out, "  ✗ %s — %s\n", s.Name(), err)
			printSummary(ctx.Out, results)
			return fmt.Errorf("step %q failed: %w", s.Name(), err)
		}
		results = append(results, stepOutcome{name: s.Name(), result: r})
		symbol := symbolFor(r.Status)
		fmt.Fprintf(ctx.Out, "  %s %s — %s\n", symbol, s.Name(), r.Message)
	}

	printSummary(ctx.Out, results)
	return nil
}

type stepOutcome struct {
	name   string
	result Result
}

func symbolFor(s Status) string {
	switch s {
	case StatusOK:
		return "✓"
	case StatusUpdated:
		return "✎"
	case StatusSkipped:
		return "↷"
	case StatusFailed:
		return "✗"
	default:
		return "·"
	}
}

func printSummary(w io.Writer, outcomes []stepOutcome) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Summary:")
	for _, o := range outcomes {
		fmt.Fprintf(w, "  %s %s — %s\n", symbolFor(o.result.Status), o.name, o.result.Message)
	}
}
