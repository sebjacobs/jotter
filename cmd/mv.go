package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var mvCmd = &cobra.Command{
	Use:   "mv <old-project> <new-project>",
	Short: "Rename a project's logs",
	Long: "Rename a project's log directory in the data repo and commit the move.\n\n" +
		"A project's jotter name is the basename of its git toplevel, so renaming the\n" +
		"project directory on disk orphans its logs under the old name. Run this to\n" +
		"carry them over: logs/<old-project> is renamed to logs/<new-project> and the\n" +
		"rename is committed (locally — like every entry, the push to the remote is left to the background timer).",
	Args:              cobra.ExactArgs(2),
	RunE:              runMv,
	ValidArgsFunction: completeMvArgs,
}

func init() {
	rootCmd.AddCommand(mvCmd)
}

func runMv(_ *cobra.Command, args []string) error {
	oldProject, newProject := args[0], args[1]

	if err := internal.ValidatePathComponent("project", oldProject); err != nil {
		return err
	}
	if err := internal.ValidatePathComponent("project", newProject); err != nil {
		return err
	}
	if oldProject == newProject {
		return fmt.Errorf("old and new project names are the same: %q", oldProject)
	}

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	oldRel := filepath.Join("logs", oldProject)
	newRel := filepath.Join("logs", newProject)

	if info, err := os.Stat(filepath.Join(dataDir, oldRel)); err != nil || !info.IsDir() {
		return fmt.Errorf("no logs for project %q", oldProject)
	}
	if _, err := os.Stat(filepath.Join(dataDir, newRel)); err == nil {
		return fmt.Errorf("project %q already has logs — refusing to overwrite", newProject)
	}

	if err := internal.GitMove(dataDir, oldRel, newRel); err != nil {
		return fmt.Errorf("renaming logs: %w", err)
	}
	if err := internal.GitCommitStaged(dataDir, fmt.Sprintf("rename: %s -> %s", oldRel, newRel)); err != nil {
		return fmt.Errorf("committing rename: %w", err)
	}

	fmt.Printf("Renamed project %s -> %s\n", internal.Dim(oldProject), internal.Dim(newProject))
	return nil
}

// completeMvArgs completes the old-project positional with known project names;
// the new-project positional is free text, so it offers nothing.
func completeMvArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completeProjects(cmd, args, toComplete)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
