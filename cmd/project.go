package cmd

import (
	"fmt"
	"os"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Print the current project name (basename of the git toplevel)",
	Long: `Print the current project name — the basename of the git toplevel for the
current working directory.

Intended for use in skill templates and scripts that need to pass --project to
other jotter commands, without boilerplating git plumbing each time.

Errors if the current directory is not inside a git repo.`,
	RunE: runProject,
}

func init() {
	rootCmd.AddCommand(projectCmd)
}

func runProject(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	name, err := internal.GitProjectName(cwd)
	if err != nil {
		return err
	}
	fmt.Println(name)
	return nil
}
