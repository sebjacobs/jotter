package cmd

import (
	"fmt"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var branchMvCmd = &cobra.Command{
	Use:   "mv <old> <new>",
	Short: "Rename a branch's logs",
	Long: "Rename a branch's log file within the data repo and commit the move.\n\n" +
		"Logs are stored per branch as logs/<project>/<branch>.jsonl, so renaming a\n" +
		"branch with `git branch -m` orphans its history under the old name. A write\n" +
		"from the renamed branch heals this automatically; run this when you won't be\n" +
		"writing to the branch again, or to migrate logs by hand. The branch's id\n" +
		"sidecar moves with it. --project defaults to the basename of the git toplevel.",
	Args:              cobra.ExactArgs(2),
	RunE:              runBranchMv,
	ValidArgsFunction: completeBranchMvArgs,
}

func init() {
	branchMvCmd.Flags().String("project", "", "Project name (default: basename of the git toplevel)")
	_ = branchMvCmd.RegisterFlagCompletionFunc("project", completeProjects)
	branchCmd.AddCommand(branchMvCmd)
}

func runBranchMv(cmd *cobra.Command, args []string) error {
	old, newName := args[0], args[1]
	if old == newName {
		return fmt.Errorf("old and new branch names are the same: %q", old)
	}

	project, err := resolveProject(cmd)
	if err != nil {
		return err
	}

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	if err := internal.MoveBranchLogs(dataDir, project, old, newName); err != nil {
		return err
	}

	fmt.Printf("Renamed branch logs %s -> %s\n", internal.Dim(old), internal.Dim(newName))
	return nil
}

// completeBranchMvArgs offers existing branch logfile names for the old
// positional and nothing for the free-text new one.
func completeBranchMvArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	project, err := resolveProject(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeBranchesFor(project), cobra.ShellCompDirectiveNoFileComp
}
