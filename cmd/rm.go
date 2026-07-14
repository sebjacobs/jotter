package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var rmForce bool

var rmCmd = &cobra.Command{
	Use:     "rm <id>",
	Aliases: []string{"revert"},
	Short:   "Remove an entry by reverting the commit that created it",
	Long: "Remove a single log entry, identified by the short commit id shown in\n" +
		"`jotter write`, `jotter tail`, and `jotter search`.\n\n" +
		"Because the log is append-only, removal is a git revert, not a rewrite: a\n" +
		"new commit drops the entry's line, so the history stays intact and the\n" +
		"removal is itself reversible. Like every jotter write it commits locally —\n" +
		"the push to the remote is left to the background timer.\n\n" +
		"Only entry commits can be reverted this way; ids that resolve to a rename,\n" +
		"relocation, or the repo's init commit are refused.",
	Args: cobra.ExactArgs(1),
	RunE: runRm,
}

func init() {
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "skip the confirmation prompt")
	rootCmd.AddCommand(rmCmd)
}

func runRm(_ *cobra.Command, args []string) error {
	id := args[0]

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	sha, err := internal.GitResolveCommit(dataDir, id)
	if err != nil {
		return fmt.Errorf("unknown entry id %q", id)
	}

	subject, err := internal.GitCommitSubject(dataDir, sha)
	if err != nil {
		return err
	}
	entry, rawLine, err := entryFromCommit(dataDir, sha, subject)
	if err != nil {
		return err
	}

	if !rmForce {
		fmt.Printf("Remove this entry?\n\n%s\n\n[y/N] ", internal.FormatEntryWithID(entry, internal.ShortSHA(sha)))
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "y", "yes":
		default:
			fmt.Println("Aborted.")
			return nil
		}
	}

	rel, err := internal.RemoveEntryLine(dataDir, rawLine)
	if err != nil {
		return fmt.Errorf("removing entry: %w", err)
	}
	msg := fmt.Sprintf("remove: %s (%s)", strings.TrimPrefix(subject, "session: "), internal.ShortSHA(sha))
	if err := internal.GitCommit(dataDir, rel, msg); err != nil {
		return fmt.Errorf("committing removal: %w", err)
	}

	removed, _ := internal.GitHeadSHA(dataDir)
	fmt.Printf("Removed %s entry %s (in %s)\n",
		internal.ColorType(entry.Type), internal.Dim(internal.ShortSHA(sha)), internal.Dim(removed))
	return nil
}

// entryFromCommit reconstructs the entry a commit created and confirms the
// commit is an entry write — not a rename, relocation, or the init commit.
// It requires the jotter "session: …" subject and exactly one added line that
// parses as a valid entry, returning that entry and its raw JSONL line. Other
// added lines (the branch-id sidecar written alongside a branch's first entry)
// don't parse as entries and are ignored, so a first entry is still removable.
func entryFromCommit(dataDir, sha, subject string) (internal.Entry, string, error) {
	if !strings.HasPrefix(subject, "session: ") {
		return internal.Entry{}, "", fmt.Errorf("%s is not a jotter entry commit (%q)", internal.ShortSHA(sha), subject)
	}
	added, err := internal.GitAddedLines(dataDir, sha)
	if err != nil {
		return internal.Entry{}, "", err
	}
	var entry internal.Entry
	var rawLine string
	found := 0
	for _, line := range added {
		e, parseErr := internal.ParseEntry(line)
		if parseErr != nil || !internal.IsValidEntryType(e.Type) {
			continue
		}
		entry, rawLine = e, line
		found++
	}
	if found != 1 {
		return internal.Entry{}, "", fmt.Errorf("%s does not add exactly one entry — refusing to remove", internal.ShortSHA(sha))
	}
	return entry, rawLine, nil
}
