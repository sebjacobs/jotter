package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

func completeProjects(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	dataDir, err := internal.GetDataDir()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	entries, err := os.ReadDir(filepath.Join(dataDir, "logs"))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func completeBranches(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	project, _ := cmd.Flags().GetString("project")
	if project == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	dataDir, err := internal.GetDataDir()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	matches, _ := filepath.Glob(filepath.Join(dataDir, "logs", project, "*.jsonl"))
	var names []string
	for _, m := range matches {
		names = append(names, internal.UnsanitiseBranch(strings.TrimSuffix(filepath.Base(m), ".jsonl")))
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func completeTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return internal.ValidEntryTypes, cobra.ShellCompDirectiveNoFileComp
}

