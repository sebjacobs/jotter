package cmd

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Push pending entries to the remote",
	Long: "Reconcile a data repo with its remote: fetch, rebase local entries " +
		"on top of any remote commits, then push. Use this to force a push now " +
		"rather than waiting for the background timer, or to recover when entries " +
		"are committed locally but not yet pushed (offline, or the remote moved on).\n\n" +
		"With --all, sync every registered data repo that has a remote — this is " +
		"what the launchd timer runs.",
	RunE: runSync,
}

func init() {
	syncCmd.Flags().Bool("all", false, "Sync every registered data repo that has a remote")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	all, _ := cmd.Flags().GetBool("all")
	if all {
		return runSyncAll(out)
	}

	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}
	if !internal.GitHasRemote(dataDir) {
		return fmt.Errorf("no git remote configured for the data repo — run `jotter setup` to add one")
	}
	return syncDataDir(out, dataDir)
}

// runSyncAll syncs every registered data repo that has a remote. Repos without
// a remote are skipped (a session-logger can legitimately run remote-less). One
// repo's failure doesn't abort the rest — failures are collected and reported,
// and the command only errors when every repo it attempted failed.
func runSyncAll(out io.Writer) error {
	dirs, err := internal.RegisteredDataDirs()
	if err != nil {
		return err
	}

	stamp := internal.Dim("[" + time.Now().Format("2006-01-02 15:04:05") + "]")
	_, _ = fmt.Fprintf(out, "%s syncing %s\n", stamp, repos(len(dirs)))
	if len(dirs) == 0 {
		_, _ = fmt.Fprintln(out, "No data repos registered yet — write an entry first")
		return nil
	}

	var synced, skipped, failed int
	for _, dir := range dirs {
		name := filepath.Base(dir)
		if !internal.GitHasRemote(dir) {
			skipped++
			_, _ = fmt.Fprintf(out, "%s %s — no remote, skipped\n", internal.Dim("·"), internal.Dim(name))
			continue
		}
		_, _ = fmt.Fprintf(out, "%s %s\n", internal.Bold("→"), name)
		if err := syncDataDir(indented(out), dir); err != nil {
			failed++
			_, _ = fmt.Fprintf(out, "  %s %v\n", internal.Dim("failed:"), err)
			continue
		}
		synced++
	}

	_, _ = fmt.Fprintf(out, "\nSynced %s, skipped %d without a remote, %d failed\n",
		internal.Bold(repos(synced)), skipped, failed)
	if failed > 0 && synced == 0 {
		return fmt.Errorf("all %d push(es) failed", failed)
	}
	return nil
}

// syncDataDir reconciles one data repo with its remote: fetch, rebase local
// entries on top of any remote commits, then push. The caller guarantees the
// repo has a remote.
func syncDataDir(out io.Writer, dataDir string) error {
	if err := internal.GitFetch(dataDir); err != nil {
		return fmt.Errorf("could not reach the remote (are you online?): %w", err)
	}

	ahead, behind, hasUpstream, err := internal.GitAheadBehind(dataDir)
	if err != nil {
		return err
	}

	if hasUpstream && behind > 0 {
		_, _ = fmt.Fprintf(out, "Remote is %s ahead — rebasing local entries on top\n",
			internal.Dim(commits(behind)))
		if err := internal.GitPullRebase(dataDir); err != nil {
			return fmt.Errorf("could not rebase onto the remote (resolve conflicts in %s, then sync again): %w", dataDir, err)
		}
		ahead, _, _, err = internal.GitAheadBehind(dataDir)
		if err != nil {
			return err
		}
	}

	if hasUpstream && ahead == 0 {
		_, _ = fmt.Fprintln(out, "Already in sync with the remote")
		return nil
	}

	if err := internal.GitPush(dataDir); err != nil {
		return err
	}
	if hasUpstream {
		_, _ = fmt.Fprintf(out, "Pushed %s to the remote\n", internal.Bold(commits(ahead)))
	} else {
		_, _ = fmt.Fprintln(out, "Pushed local entries to the remote")
	}
	return nil
}

func commits(n int) string {
	if n == 1 {
		return "1 commit"
	}
	return fmt.Sprintf("%d commits", n)
}

func repos(n int) string {
	if n == 1 {
		return "1 repo"
	}
	return fmt.Sprintf("%d repos", n)
}

// indented wraps w so each line written through it is prefixed with two spaces,
// nesting a repo's sync output under its `→ name` header in `--all` mode. The
// single-repo path writes to the bare writer and stays flush-left.
func indented(w io.Writer) io.Writer {
	return &indentWriter{w: w, atLineStart: true}
}

type indentWriter struct {
	w           io.Writer
	atLineStart bool
}

func (iw *indentWriter) Write(p []byte) (int, error) {
	for _, line := range bytes.SplitAfter(p, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		if iw.atLineStart {
			if _, err := iw.w.Write([]byte("  ")); err != nil {
				return 0, err
			}
		}
		if _, err := iw.w.Write(line); err != nil {
			return 0, err
		}
		iw.atLineStart = line[len(line)-1] == '\n'
	}
	return len(p), nil
}
