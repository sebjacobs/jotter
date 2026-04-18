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

	if branch != "" && project == "" {
		fmt.Fprintln(os.Stderr, "--branch requires --project")
		os.Exit(1)
	}

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
		return lsEntries(dataDir, project, branch)
	}
	if project != "" {
		return lsBranches(logsDir, project)
	}
	return lsProjects(logsDir)
}

func lsEntries(dataDir, project, branch string) error {
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

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		t, _ := time.Parse(internal.TimestampFormat, e.Timestamp)
		ts := t.Format("2006-01-02 15:04")
		fmt.Printf("%s  %-10s  %s\n", internal.Dim(ts), internal.ColorType(e.Type), entryTitle(e.Content))
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

func lsBranches(logsDir, project string) error {
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
		bi := branchInfo{name: name}
		if len(entries) > 0 {
			bi.count = len(entries)
			bi.lastTS = entries[len(entries)-1].Timestamp
			t, _ := time.Parse(internal.TimestampFormat, bi.lastTS)
			bi.lastDate = t.Format(internal.DateFormat)
		}
		branches = append(branches, bi)
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

func lsProjects(logsDir string) error {
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
			if err != nil || len(logEntries) == 0 {
				continue
			}
			ts := logEntries[len(logEntries)-1].Timestamp
			if ts > pi.lastTS {
				pi.lastTS = ts
				t, _ := time.Parse(internal.TimestampFormat, ts)
				pi.lastDate = t.Format(internal.DateFormat)
			}
		}
		projects = append(projects, pi)
	}

	if len(projects) == 0 {
		fmt.Fprintln(os.Stderr, "No projects found")
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
