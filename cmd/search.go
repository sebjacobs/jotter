package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [term]",
	Short: "Search entries by content",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSearch,
}

func init() {
	searchCmd.Flags().String("project", "", "Scope to this project (default: current git project)")
	searchCmd.Flags().String("branch", "", "Scope to this branch (default: current git branch)")
	searchCmd.Flags().BoolP("all", "a", false, "Search every project and branch, ignoring the current repo")
	searchCmd.Flags().String("since", "", "Filter entries from this date/time (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, inclusive)")
	searchCmd.Flags().String("until", "", "Filter entries up to this date/time (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, inclusive)")
	searchCmd.Flags().String("type", "", "Filter by entry type")
	addFormatFlag(searchCmd)
	_ = searchCmd.RegisterFlagCompletionFunc("project", completeProjects)
	_ = searchCmd.RegisterFlagCompletionFunc("branch", completeBranches)
	_ = searchCmd.RegisterFlagCompletionFunc("type", completeTypes)
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	scope, err := resolveScope(cmd, true)
	if err != nil {
		return err
	}
	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")
	entryType, _ := cmd.Flags().GetString("type")
	format, err := outputFormat(cmd)
	if err != nil {
		return err
	}

	var term string
	if len(args) > 0 {
		term = strings.ToLower(args[0])
	}

	sinceTime, untilTime, err := parseWindow(since, until)
	if err != nil {
		return err
	}

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	logsDir := filepath.Join(dataDir, "logs")
	paths, err := scope.Paths(dataDir)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "No matching log files found")
		os.Exit(1)
	}

	var results []string
	var jsonResults []entryJSON
	for _, path := range paths {
		entries, err := internal.ReadEntries(path)
		if err != nil {
			continue
		}

		rel, _ := filepath.Rel(logsDir, path)
		project := filepath.Dir(rel)
		branch := internal.UnsanitiseBranch(strings.TrimSuffix(filepath.Base(rel), ".jsonl"))
		shas := entrySHAs(dataDir, path, len(entries))

		for i, entry := range entries {
			if !inWindow(entry.Timestamp, sinceTime, untilTime) {
				continue
			}

			if entryType != "" && entry.Type != entryType {
				continue
			}

			if term != "" {
				searchable := strings.ToLower(entry.Content)
				if entry.Next != "" {
					searchable += " " + strings.ToLower(entry.Next)
				}
				if !strings.Contains(searchable, term) {
					continue
				}
			}

			id := ""
			if shas != nil {
				id = shas[i]
			}
			if format == formatJSON {
				jsonResults = append(jsonResults, newEntryJSON(entry, project, branch, id))
				continue
			}
			results = append(results, fmt.Sprintf("%s\n%s", internal.Dim("["+rel+"]"), internal.FormatEntryWithID(entry, id)))
		}
	}

	if format == formatJSON && len(jsonResults) > 0 {
		return printJSON(jsonResults)
	}
	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No matching entries found")
		os.Exit(1)
	}

	fmt.Println(strings.Join(results, "\n\n"))
	return nil
}
