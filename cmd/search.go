package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	searchCmd.Flags().String("project", "", "Scope to this project")
	searchCmd.Flags().String("branch", "", "Scope to this branch")
	searchCmd.Flags().String("since", "", "Filter entries from this date (YYYY-MM-DD)")
	searchCmd.Flags().String("type", "", "Filter by entry type")
	_ = searchCmd.RegisterFlagCompletionFunc("project", completeProjects)
	_ = searchCmd.RegisterFlagCompletionFunc("branch", completeBranches)
	_ = searchCmd.RegisterFlagCompletionFunc("type", completeTypes)
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	project, _ := cmd.Flags().GetString("project")
	branch, _ := cmd.Flags().GetString("branch")
	since, _ := cmd.Flags().GetString("since")
	entryType, _ := cmd.Flags().GetString("type")

	var term string
	if len(args) > 0 {
		term = strings.ToLower(args[0])
	}

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	logsDir := filepath.Join(dataDir, "logs")
	paths := internal.CollectPaths(dataDir, project, branch)
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "No matching log files found")
		os.Exit(1)
	}

	var sinceTime time.Time
	if since != "" {
		sinceTime, err = time.Parse(internal.DateFormat, since)
		if err != nil {
			return fmt.Errorf("invalid --since date: %w", err)
		}
	}

	var results []string
	for _, path := range paths {
		entries, err := internal.ReadEntries(path)
		if err != nil {
			continue
		}

		rel, _ := filepath.Rel(logsDir, path)

		for _, entry := range entries {
			if !sinceTime.IsZero() {
				entryTime, _ := time.Parse(internal.TimestampFormat, entry.Timestamp)
				if entryTime.Before(sinceTime) {
					continue
				}
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

			results = append(results, fmt.Sprintf("%s\n%s", internal.Dim("["+rel+"]"), internal.FormatEntry(entry)))
		}
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No matching entries found")
		os.Exit(1)
	}

	fmt.Println(strings.Join(results, "\n\n"))
	return nil
}
