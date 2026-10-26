package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

const (
	formatMarkdown = "md"
	formatJSON     = "json"
)

type entryJSON struct {
	ID        string `json:"id,omitempty"`
	Project   string `json:"project"`
	Branch    string `json:"branch"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Next      string `json:"next,omitempty"`
}

type projectJSON struct {
	Name string `json:"name"`
	Last string `json:"last,omitempty"`
}

type branchJSON struct {
	Name    string `json:"name"`
	Entries int    `json:"entries"`
	Last    string `json:"last,omitempty"`
}

type entryTitleJSON struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Title     string `json:"title"`
}

func newEntryJSON(e internal.Entry, project, branch, id string) entryJSON {
	return entryJSON{
		ID:        id,
		Project:   project,
		Branch:    branch,
		Timestamp: e.Timestamp,
		Type:      e.Type,
		Content:   e.Content,
		Next:      e.Next,
	}
}

func addFormatFlag(cmd *cobra.Command) {
	cmd.Flags().String("format", formatMarkdown, "Output format: md or json")
	_ = cmd.RegisterFlagCompletionFunc("format", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{formatMarkdown, formatJSON}, cobra.ShellCompDirectiveNoFileComp
	})
}

func outputFormat(cmd *cobra.Command) (string, error) {
	format, _ := cmd.Flags().GetString("format")
	switch format {
	case formatMarkdown, formatJSON:
		return format, nil
	}
	return "", fmt.Errorf("invalid --format %q: must be md or json", format)
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
