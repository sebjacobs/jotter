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
	_ = lsCmd.RegisterFlagCompletionFunc("project", completeProjects)
	rootCmd.AddCommand(lsCmd)
}

type branchInfo struct {
	name    string
	count   int
	lastRel string
	lastTS  string
}

type projectInfo struct {
	name    string
	lastRel string
	lastTS  string
}

func relativeTime(ts string) string {
	t, err := time.Parse(internal.TimestampFormat, ts)
	if err != nil {
		return ts
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 14*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 8*7*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	default:
		return t.Format("2006-01-02 15:04")
	}
}

func runLs(cmd *cobra.Command, args []string) error {
	project, _ := cmd.Flags().GetString("project")

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	logsDir := filepath.Join(dataDir, "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "No logs directory found")
		os.Exit(1)
	}

	if project != "" {
		return lsBranches(logsDir, project)
	}
	return lsProjects(logsDir)
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
			bi.lastRel = relativeTime(bi.lastTS)
		}
		branches = append(branches, bi)
	}

	slices.SortFunc(branches, func(a, b branchInfo) int {
		return strings.Compare(b.lastTS, a.lastTS)
	})

	for _, b := range branches {
		if b.lastRel != "" {
			fmt.Printf("%s  %s\n", internal.Bold(b.name), internal.Dim(fmt.Sprintf("(%d entries, last: %s)", b.count, b.lastRel)))
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
				pi.lastRel = relativeTime(ts)
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
		if p.lastRel != "" {
			fmt.Printf("%s  %s\n", internal.Bold(p.name), internal.Dim(fmt.Sprintf("(last: %s)", p.lastRel)))
		} else {
			fmt.Println(internal.Bold(p.name))
		}
	}
	return nil
}
