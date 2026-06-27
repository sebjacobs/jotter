package cmd

import (
	"os"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

// resolveProject returns the --project flag value, falling back to the basename
// of the git toplevel for the current directory when the flag is empty.
func resolveProject(cmd *cobra.Command) (string, error) {
	if project, _ := cmd.Flags().GetString("project"); project != "" {
		return project, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return internal.GitProjectName(cwd)
}

// resolveBranch returns the --branch flag value, falling back to the current
// git branch for the current directory when the flag is empty.
func resolveBranch(cmd *cobra.Command) (string, error) {
	if branch, _ := cmd.Flags().GetString("branch"); branch != "" {
		return branch, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return internal.GitCurrentBranch(cwd)
}
