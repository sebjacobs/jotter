package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List projects or branches",
	RunE:  runLs,
}

func init() {
	lsCmd.Flags().String("project", "", "List branches for this project")
	lsCmd.Flags().String("branch", "", "List entries for this branch (requires --project)")
	lsCmd.Flags().String("since", "", "Only include entries from this date/time (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, inclusive)")
	lsCmd.Flags().String("until", "", "Only include entries up to this date/time (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, inclusive)")
	_ = lsCmd.RegisterFlagCompletionFunc("project", completeProjects)
	_ = lsCmd.RegisterFlagCompletionFunc("branch", completeBranches)
	rootCmd.AddCommand(lsCmd)
}

type branchInfo struct {
	name     string
	count    int
	lastDate string
	lastTS   string
}

type projectInfo struct {
	name     string
	lastDate string
	lastTS   string
}

func runLs(cmd *cobra.Command, args []string) error {
	project, _ := cmd.Flags().GetString("project")
	branch, _ := cmd.Flags().GetString("branch")
	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")

	if branch != "" && project == "" {
		fmt.Fprintln(os.Stderr, "--branch requires --project")
		os.Exit(1)
	}

	sinceTime, untilTime, err := parseWindow(since, until)
	if err != nil {
		return err
	}
	windowActive := !sinceTime.IsZero() || !untilTime.IsZero()

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	logsDir := filepath.Join(dataDir, "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "No logs directory found")
		os.Exit(1)
	}

	if branch != "" {
		return lsEntries(dataDir, project, branch, sinceTime, untilTime, windowActive)
	}
	if project != "" {
		return lsBranches(logsDir, project, sinceTime, untilTime, windowActive)
	}
	return lsProjects(logsDir, sinceTime, untilTime, windowActive)
}

func lsEntries(dataDir, project, branch string, since, until time.Time, windowActive bool) error {
	path, err := internal.JSONLPath(dataDir, project, branch)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "No logs for %s/%s\n", project, branch)
		os.Exit(1)
	}
	entries, err := internal.ReadEntries(path)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "No entries for %s/%s\n", project, branch)
		os.Exit(1)
	}

	printed := 0
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if !inWindow(e.Timestamp, since, until) {
			continue
		}
		t, _ := time.Parse(internal.TimestampFormat, e.Timestamp)
		ts := t.Format("2006-01-02 15:04")
		fmt.Printf("%s  %-10s  %s\n", internal.Dim(ts), internal.ColorType(e.Type), entryTitle(e.Content))
		printed++
	}
	if printed == 0 {
		if windowActive {
			fmt.Fprintf(os.Stderr, "No entries for %s/%s in window\n", project, branch)
		} else {
			fmt.Fprintf(os.Stderr, "No entries for %s/%s\n", project, branch)
		}
		os.Exit(1)
	}
	return nil
}

// entryTitle extracts a short single-line title from the first non-empty
// line of an entry's content, stripping leading markdown markers.
func entryTitle(content string) string {
	for raw := range strings.SplitSeq(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		line = strings.TrimLeft(line, "#>- \t")
		line = strings.ReplaceAll(line, "**", "")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		const maxLen = 100
		if len([]rune(line)) > maxLen {
			line = string([]rune(line)[:maxLen]) + "…"
		}
		return line
	}
	return ""
}

func lsBranches(logsDir, project string, since, until time.Time, windowActive bool) error {
	projectDir := filepath.Join(logsDir, project)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "No logs for project %s\n", project)
		os.Exit(1)
	}

	matches, _ := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
	var branches []branchInfo
	for _, path := range matches {
		name := internal.UnsanitiseBranch(strings.TrimSuffix(filepath.Base(path), ".jsonl"))
		entries, err := internal.ReadEntries(path)
		if err != nil {
			continue
		}
		bi := branchInfoFromEntries(name, entries, since, until)
		if windowActive && bi.count == 0 {
			continue
		}
		branches = append(branches, bi)
	}

	if windowActive && len(branches) == 0 {
		fmt.Fprintf(os.Stderr, "No branches for project %s in window\n", project)
		os.Exit(1)
	}

	slices.SortFunc(branches, func(a, b branchInfo) int {
		return strings.Compare(b.lastTS, a.lastTS)
	})

	for _, b := range branches {
		if b.lastDate != "" {
			fmt.Printf("%s  %s\n", internal.Bold(b.name), internal.Dim(fmt.Sprintf("(%d entries, last: %s)", b.count, b.lastDate)))
		} else {
			fmt.Println(internal.Bold(b.name))
		}
	}
	return nil
}

func branchInfoFromEntries(name string, entries []internal.Entry, since, until time.Time) branchInfo {
	bi := branchInfo{name: name}
	for _, e := range entries {
		if !inWindow(e.Timestamp, since, until) {
			continue
		}
		bi.count++
		if e.Timestamp > bi.lastTS {
			bi.lastTS = e.Timestamp
			t, _ := time.Parse(internal.TimestampFormat, bi.lastTS)
			bi.lastDate = t.Format("2006-01-02 15:04")
		}
	}
	return bi
}

func lsProjects(logsDir string, since, until time.Time, windowActive bool) error {
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return err
	}

	var projects []projectInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(logsDir, entry.Name(), "*.jsonl"))
		pi := projectInfo{name: entry.Name()}
		for _, path := range matches {
			logEntries, err := internal.ReadEntries(path)
			if err != nil {
				continue
			}
			for _, e := range logEntries {
				if !inWindow(e.Timestamp, since, until) {
					continue
				}
				if e.Timestamp > pi.lastTS {
					pi.lastTS = e.Timestamp
					t, _ := time.Parse(internal.TimestampFormat, e.Timestamp)
					pi.lastDate = t.Format("2006-01-02 15:04")
				}
			}
		}
		if windowActive && pi.lastTS == "" {
			continue
		}
		projects = append(projects, pi)
	}

	if len(projects) == 0 {
		if windowActive {
			fmt.Fprintln(os.Stderr, "No projects with entries in window")
		} else {
			fmt.Fprintln(os.Stderr, "No projects found")
		}
		os.Exit(1)
	}

	slices.SortFunc(projects, func(a, b projectInfo) int {
		return strings.Compare(b.lastTS, a.lastTS)
	})

	for _, p := range projects {
		if p.lastDate != "" {
			fmt.Printf("%s  %s\n", internal.Bold(p.name), internal.Dim(fmt.Sprintf("(last: %s)", p.lastDate)))
		} else {
			fmt.Println(internal.Bold(p.name))
		}
	}
	return nil
}
