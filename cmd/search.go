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
	searchCmd.Flags().String("since", "", "Filter entries from this date/time (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, inclusive)")
	searchCmd.Flags().String("until", "", "Filter entries up to this date/time (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, inclusive)")
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
	until, _ := cmd.Flags().GetString("until")
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
	paths, err := internal.CollectPaths(dataDir, project, branch)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "No matching log files found")
		os.Exit(1)
	}

	var sinceTime time.Time
	if since != "" {
		sinceTime, err = parseBoundary(since, false)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
	}

	var untilTime time.Time
	if until != "" {
		untilTime, err = parseBoundary(until, true)
		if err != nil {
			return fmt.Errorf("invalid --until value: %w", err)
		}
	}

	if !sinceTime.IsZero() && !untilTime.IsZero() && untilTime.Before(sinceTime) {
		return fmt.Errorf("--until must not be earlier than --since")
	}

	var results []string
	for _, path := range paths {
		entries, err := internal.ReadEntries(path)
		if err != nil {
			continue
		}

		rel, _ := filepath.Rel(logsDir, path)

		for _, entry := range entries {
			if !sinceTime.IsZero() || !untilTime.IsZero() {
				entryTime, _ := time.Parse(internal.TimestampFormat, entry.Timestamp)
				if !sinceTime.IsZero() && entryTime.Before(sinceTime) {
					continue
				}
				if !untilTime.IsZero() && entryTime.After(untilTime) {
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

// parseBoundary parses either a date (YYYY-MM-DD) or full timestamp
// (YYYY-MM-DDTHH:MM:SS). For date-only values, endOfDay=true promotes
// the result to 23:59:59 so --until <date> is inclusive of that day.
func parseBoundary(s string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(internal.TimestampFormat, s); err == nil {
		return t, nil
	}
	t, err := time.Parse(internal.DateFormat, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, got %q", s)
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Second)
	}
	return t, nil
}
