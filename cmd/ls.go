package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
		name := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		entries, err := internal.ReadEntries(path)
		if err != nil {
			continue
		}
		bi := branchInfo{name: name}
		if len(entries) > 0 {
			bi.count = len(entries)
			bi.lastTS = entries[len(entries)-1].Timestamp
			t, _ := time.Parse("2006-01-02T15:04:05", bi.lastTS)
			bi.lastDate = t.Format("2006-01-02")
		}
		branches = append(branches, bi)
	}

	sort.Slice(branches, func(i, j int) bool {
		return branches[i].lastTS > branches[j].lastTS
	})

	for _, b := range branches {
		if b.lastDate != "" {
			fmt.Printf("%s  (%d entries, last: %s)\n", b.name, b.count, b.lastDate)
		} else {
			fmt.Println(b.name)
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
				t, _ := time.Parse("2006-01-02T15:04:05", ts)
				pi.lastDate = t.Format("2006-01-02")
			}
		}
		projects = append(projects, pi)
	}

	if len(projects) == 0 {
		fmt.Fprintln(os.Stderr, "No projects found")
		os.Exit(1)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].lastTS > projects[j].lastTS
	})

	for _, p := range projects {
		if p.lastDate != "" {
			fmt.Printf("%s  (last: %s)\n", p.name, p.lastDate)
		} else {
			fmt.Println(p.name)
		}
	}
	return nil
}
