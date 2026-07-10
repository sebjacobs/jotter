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

// resolveScope builds the query Scope for ls and search, defaulting to the repo
// you're standing in the way gwt/proj do:
//
//   - --all forces AllScope, ignoring the current repo.
//   - An explicit --project is honoured; --branch (if any) narrows within it,
//     otherwise all of that project's branches are in scope.
//   - With neither flag, project defaults to the cwd git project. Branch is then
//     defaulted to the cwd git branch only when defaultBranch is true (search
//     narrows to the current branch; ls widens to the project's branch list).
//   - Outside a git repo there is nothing to default to, so scope falls back to
//     AllScope — unless --branch was given without a resolvable project, which is
//     ambiguous and errors.
func resolveScope(cmd *cobra.Command, defaultBranch bool) (internal.Scope, error) {
	if all, _ := cmd.Flags().GetBool("all"); all {
		return internal.AllScope{}, nil
	}

	project, _ := cmd.Flags().GetString("project")
	branch, _ := cmd.Flags().GetString("branch")

	if project == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		p, perr := internal.GitProjectName(cwd)
		if perr != nil {
			if branch != "" {
				return nil, fmt.Errorf("--branch requires --project when not inside a git repository")
			}
			return internal.AllScope{}, nil
		}
		project = p
		if branch == "" && defaultBranch {
			if b, berr := internal.GitCurrentBranch(cwd); berr == nil {
				branch = b
			}
		}
	}

	if branch == "" {
		return internal.ProjectScope{Project: project}, nil
	}
	return internal.BranchScope{Project: project, Branch: branch}, nil
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
