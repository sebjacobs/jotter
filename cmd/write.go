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
	writeCmd.Flags().String("project", "", "Project name (default: basename of the git toplevel)")
	writeCmd.Flags().String("branch", "", "Branch name (default: current git branch)")
	writeCmd.Flags().String("type", "", "Entry type: start, checkpoint, note, break, stop, handover (finish is a legacy alias for stop) (required)")
	writeCmd.Flags().String("content", "", "Entry content (required)")
	writeCmd.Flags().String("next", "", "Next task description")
	_ = writeCmd.MarkFlagRequired("type")
	_ = writeCmd.MarkFlagRequired("content")
	_ = writeCmd.RegisterFlagCompletionFunc("project", completeProjects)
	_ = writeCmd.RegisterFlagCompletionFunc("branch", completeBranches)
	_ = writeCmd.RegisterFlagCompletionFunc("type", completeTypes)
	rootCmd.AddCommand(writeCmd)
}

func runWrite(cmd *cobra.Command, args []string) error {
	entryType, _ := cmd.Flags().GetString("type")
	content, _ := cmd.Flags().GetString("content")
	next, _ := cmd.Flags().GetString("next")

	if !internal.IsValidEntryType(entryType) {
		return fmt.Errorf("invalid entry type %q: must be one of start, checkpoint, note, break, stop, handover (finish is a legacy alias for stop)", entryType)
	}

	project, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	branch, err := resolveBranch(cmd)
	if err != nil {
		return err
	}

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	if regErr := internal.RegisterDataDir(dataDir); regErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not register data dir for background push: %v\n", regErr)
	}

	path, err := internal.JSONLPath(dataDir, project, branch)
	if err != nil {
		return err
	}

	trackSidecar := false
	if cwd, cwdErr := os.Getwd(); cwdErr == nil && onBranch(cwd, project, branch) {
		if reconciled, warn := internal.ReconcileBranch(dataDir, cwd, project, branch); warn != nil {
			fmt.Fprintf(os.Stderr, "Warning: branch tracking skipped: %v\n", warn)
		} else {
			path = reconciled
			trackSidecar = true
		}
	}

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
	if trackSidecar {
		if sidecar, scErr := internal.SidecarPath(dataDir, project, branch); scErr == nil {
			if _, statErr := os.Stat(sidecar); statErr == nil {
				_ = internal.GitAdd(dataDir, sidecar)
			}
		}
	}
	if err := internal.GitCommit(dataDir, path, commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	fmt.Printf("Wrote %s entry to %s\n", internal.ColorType(entryType), internal.Dim(rel))
	return nil
}
