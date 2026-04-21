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
	searchCmd.Flags().String("project", "", "Scope to this project")
	searchCmd.Flags().String("branch", "", "Scope to this branch")
	searchCmd.Flags().String("since", "", "Filter entries from this date/time (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, inclusive)")
	searchCmd.Flags().String("until", "", "Filter entries up to this date/time (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, inclusive)")
	searchCmd.Flags().String("type", "", "Filter by entry type")
	searchCmd.Flags().Int("limit", 0, "Maximum number of entries to return (0 = no limit)")
	searchCmd.Flags().Int("offset", 0, "Number of entries to skip")
	_ = searchCmd.RegisterFlagCompletionFunc("project", completeProjects)
	_ = searchCmd.RegisterFlagCompletionFunc("branch", completeBranches)
	_ = searchCmd.RegisterFlagCompletionFunc("type", completeTypes)
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	project, _ := cmd.Flags().GetString("project")
	branch, _ := cmd.Flags().GetString("branch")
	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")
	entryType, _ := cmd.Flags().GetString("type")
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")

	if limit < 0 {
		return fmt.Errorf("--limit must be >= 0")
	}
	if offset < 0 {
		return fmt.Errorf("--offset must be >= 0")
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
	paths, err := internal.CollectPaths(dataDir, project, branch)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "No matching log files found")
		os.Exit(1)
	}

	var results []string
	for _, path := range paths {
		entries, err := internal.ReadEntries(path)
		if err != nil {
			continue
		}

		rel, _ := filepath.Rel(logsDir, path)

		for _, entry := range entries {
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

			results = append(results, fmt.Sprintf("%s\n%s", internal.Dim("["+rel+"]"), internal.FormatEntry(entry)))
		}
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No matching entries found")
		os.Exit(1)
	}

	total := len(results)
	if offset >= total {
		fmt.Fprintf(os.Stderr, "offset %d exceeds %d result%s\n", offset, total, plural(total))
		os.Exit(1)
	}

	end := total
	if limit > 0 && offset+limit < total {
		end = offset + limit
	}
	page := results[offset:end]

	fmt.Println(strings.Join(page, "\n\n"))

	if limit > 0 || offset > 0 {
		fmt.Fprintln(os.Stderr, paginationFooter(offset, end, total))
	}
	return nil
}

func paginationFooter(offset, end, total int) string {
	if end >= total {
		return fmt.Sprintf("Showing %d–%d of %d (end)", offset+1, end, total)
	}
	return fmt.Sprintf("Showing %d–%d of %d — next: --offset %d", offset+1, end, total, end)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
