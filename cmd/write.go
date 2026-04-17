package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var writeCmd = &cobra.Command{
	Use:   "write",
	Short: "Append a session log entry",
	RunE:  runWrite,
}

func init() {
	writeCmd.Flags().String("project", "", "Project name (required)")
	writeCmd.Flags().String("branch", "", "Branch name (required)")
	writeCmd.Flags().String("type", "", "Entry type: start, checkpoint, note, break, finish (required)")
	writeCmd.Flags().String("content", "", "Entry content (required)")
	writeCmd.Flags().String("next", "", "Next task description")
	_ = writeCmd.MarkFlagRequired("project")
	_ = writeCmd.MarkFlagRequired("branch")
	_ = writeCmd.MarkFlagRequired("type")
	_ = writeCmd.MarkFlagRequired("content")
	_ = writeCmd.RegisterFlagCompletionFunc("project", completeProjects)
	_ = writeCmd.RegisterFlagCompletionFunc("branch", completeBranches)
	_ = writeCmd.RegisterFlagCompletionFunc("type", completeTypes)
	rootCmd.AddCommand(writeCmd)
}

func runWrite(cmd *cobra.Command, args []string) error {
	project, _ := cmd.Flags().GetString("project")
	branch, _ := cmd.Flags().GetString("branch")
	entryType, _ := cmd.Flags().GetString("type")
	content, _ := cmd.Flags().GetString("content")
	next, _ := cmd.Flags().GetString("next")

	if !internal.IsValidEntryType(entryType) {
		return fmt.Errorf("invalid entry type %q: must be one of start, checkpoint, note, break, finish", entryType)
	}

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	path := internal.JSONLPath(dataDir, project, branch)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}

	entry := internal.Entry{
		Timestamp: time.Now().Format(internal.TimestampFormat),
		Type:      entryType,
		Content:   content,
	}
	if next != "" {
		entry.Next = next
	}

	data, err := internal.MarshalJSONL(entry)
	if err != nil {
		return fmt.Errorf("marshaling entry: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	_, writeErr := fmt.Fprintf(f, "%s\n", data)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}

	rel, _ := filepath.Rel(dataDir, path)
	timestamp := entry.Timestamp
	commitMsg := fmt.Sprintf("session: %s/%s %s %s", project, branch, entryType, timestamp)
	if err := internal.GitCommit(dataDir, path, commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	if entryType == "finish" {
		if err := internal.GitPush(dataDir); err != nil {
			// Push failure is non-fatal — warn but don't fail
			fmt.Fprintf(os.Stderr, "Warning: git push failed: %v\n", err)
		}
	}

	fmt.Printf("Wrote %s entry to %s\n", internal.ColorType(entryType), internal.Dim(rel))
	return nil
}
