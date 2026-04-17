package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/sebjacobs/jotter/internal/setup"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive wizard to configure jotter for a Claude Code session",
	Long: `Interactive wizard that takes you from 'jotter installed' to '/start works'
in one flow: detects Claude Code, prompts for a data directory, initialises the
git-backed data repo, writes the .jotter config, installs session-management
skills into ~/.claude/skills/, merges the Bash(jotter:*) permission, and runs a
smoke test.

The wizard is idempotent — re-running detects existing state and updates only
what needs updating.`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("locating home directory: %w", err)
	}

	ctx := &setup.Context{
		Home:     home,
		SkillsFS: skillsFS,
		Prompter: &huhPrompter{},
		Answers:  &setup.Answers{},
		Out:      cmd.OutOrStdout(),
	}

	_, _ = fmt.Fprintln(ctx.Out, "jotter setup")
	_, _ = fmt.Fprintln(ctx.Out, "")
	return setup.Run(ctx, setup.DefaultSteps())
}

// huhPrompter is the production implementation of setup.Prompter, using
// charmbracelet/huh for actual interactive TUI prompts.
type huhPrompter struct{}

func (huhPrompter) Input(question, defaultValue string) (string, error) {
	value := defaultValue
	err := huh.NewInput().
		Title(question).
		Value(&value).
		Run()
	if err != nil {
		return "", err
	}
	return value, nil
}

func (huhPrompter) Confirm(question string, defaultYes bool) (bool, error) {
	value := defaultYes
	err := huh.NewConfirm().
		Title(question).
		Value(&value).
		Run()
	if err != nil {
		return false, err
	}
	return value, nil
}
