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
	tailCmd.Flags().String("project", "", "Project name (default: basename of the git toplevel)")
	tailCmd.Flags().String("branch", "", "Branch name (default: current git branch)")
	tailCmd.Flags().IntP("limit", "n", 1, "Number of entries to return")
	addFormatFlag(tailCmd)
	_ = tailCmd.RegisterFlagCompletionFunc("project", completeProjects)
	_ = tailCmd.RegisterFlagCompletionFunc("branch", completeBranches)
	rootCmd.AddCommand(tailCmd)
}

func runTail(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	format, err := outputFormat(cmd)
	if err != nil {
		return err
	}

	project, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	branch, err := resolveBranch(cmd)
	if err != nil {
		return err
	}

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	path, err := internal.JSONLPath(dataDir, project, branch)
	if err != nil {
		return err
	}
	entries, err := internal.ReadEntries(path)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "No log file for %s/%s\n", project, branch)
		os.Exit(1)
	}

	shas := entrySHAs(dataDir, path, len(entries))

	start := max(0, len(entries)-limit)
	tail := entries[start:]

	ids := make([]string, len(tail))
	if shas != nil {
		copy(ids, shas[start:])
	}

	if format == formatJSON {
		out := make([]entryJSON, len(tail))
		for i, e := range tail {
			out[i] = newEntryJSON(e, project, branch, ids[i])
		}
		return printJSON(out)
	}

	formatted := make([]string, len(tail))
	for i, e := range tail {
		formatted[i] = internal.FormatEntryWithID(e, ids[i])
	}
	fmt.Println(strings.Join(formatted, "\n\n"))
	return nil
}
