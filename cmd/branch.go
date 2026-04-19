package cmd

import (
	"fmt"
	"os"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Print the current git branch name",
	Long: `Print the current git branch name for the current working directory.

Intended for use in skill templates and scripts that need to pass --branch to
other jotter commands, without boilerplating git plumbing each time.

Errors if the current directory is not inside a git repo, or if HEAD is
detached.`,
	RunE: runBranch,
}

func init() {
	rootCmd.AddCommand(branchCmd)
}

func runBranch(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	name, err := internal.GitCurrentBranch(cwd)
	if err != nil {
		return err
	}
	fmt.Println(name)
	return nil
}
