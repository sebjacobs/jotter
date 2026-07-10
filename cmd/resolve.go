package cmd

import (
	"fmt"
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

// resolveScope builds the query Scope for ls and search from the --project and
// --branch flags. An empty --project means all projects; an empty --branch
// within a project means all of its branches; --branch without --project is
// ambiguous and errors.
func resolveScope(cmd *cobra.Command) (internal.Scope, error) {
	project, _ := cmd.Flags().GetString("project")
	branch, _ := cmd.Flags().GetString("branch")

	switch {
	case project == "":
		if branch != "" {
			return nil, fmt.Errorf("--branch requires --project")
		}
		return internal.AllScope{}, nil
	case branch == "":
		return internal.ProjectScope{Project: project}, nil
	default:
		return internal.BranchScope{Project: project, Branch: branch}, nil
	}
}

// onBranch reports whether cwd's repo is exactly the project and branch being
// written to — the precondition for branch-rename reconciliation. A
// cross-project or off-branch write (or one run outside a git repo) returns
// false, so reconciliation is skipped and the write behaves as it always has.
func onBranch(cwd, project, branch string) bool {
	if p, err := internal.GitProjectName(cwd); err != nil || p != project {
		return false
	}
	if b, err := internal.GitCurrentBranch(cwd); err != nil || b != branch {
		return false
	}
	return true
}
