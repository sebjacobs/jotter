package cmd

import (
	"fmt"
	"io/fs"

	"github.com/spf13/cobra"
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
	// Placeholder: prove the skills embed is wired through.
	// Full wizard implementation lands in subsequent commits.
	count, err := countEmbeddedSkills()
	if err != nil {
		return fmt.Errorf("counting embedded skills: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "jotter setup: wired up (embedded %d skills)\n", count)
	return nil
}

func countEmbeddedSkills() (int, error) {
	entries, err := fs.ReadDir(skillsFS, "skills")
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}
