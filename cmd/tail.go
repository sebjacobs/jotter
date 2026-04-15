package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var tailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Show recent entries for a branch",
	RunE:  runTail,
}

func init() {
	tailCmd.Flags().String("project", "", "Project name (required)")
	tailCmd.Flags().String("branch", "", "Branch name (required)")
	tailCmd.Flags().Int("limit", 1, "Number of entries to return")
	tailCmd.MarkFlagRequired("project")
	tailCmd.MarkFlagRequired("branch")
	rootCmd.AddCommand(tailCmd)
}

func runTail(cmd *cobra.Command, args []string) error {
	project, _ := cmd.Flags().GetString("project")
	branch, _ := cmd.Flags().GetString("branch")
	limit, _ := cmd.Flags().GetInt("limit")

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	path := internal.JSONLPath(dataDir, project, branch)
	entries, err := internal.ReadEntries(path)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "No log file for %s/%s\n", project, branch)
		os.Exit(1)
	}

	// Take the last `limit` entries
	start := len(entries) - limit
	if start < 0 {
		start = 0
	}
	tail := entries[start:]

	formatted := make([]string, len(tail))
	for i, e := range tail {
		formatted[i] = internal.FormatEntry(e)
	}
	fmt.Println(strings.Join(formatted, "\n\n"))
	return nil
}
