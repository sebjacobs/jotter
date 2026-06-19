package cmd

import (
	"fmt"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Push pending entries to the remote after a failed push",
	Long: "Reconcile the data repo with its remote: fetch, rebase local entries " +
		"on top of any remote commits, then push. Use this when a finish push " +
		"failed (offline, or the remote moved on) and entries are committed " +
		"locally but not yet pushed.",
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	if !internal.GitHasRemote(dataDir) {
		return fmt.Errorf("no git remote configured for the data repo — run `jotter setup` to add one")
	}

	if err := internal.GitFetch(dataDir); err != nil {
		return fmt.Errorf("could not reach the remote (are you online?): %w", err)
	}

	ahead, behind, hasUpstream, err := internal.GitAheadBehind(dataDir)
	if err != nil {
		return err
	}

	if hasUpstream && behind > 0 {
		fmt.Printf("Remote is %s ahead — rebasing local entries on top\n",
			internal.Dim(commits(behind)))
		if err := internal.GitPullRebase(dataDir); err != nil {
			return fmt.Errorf("could not rebase onto the remote (resolve conflicts in %s, then run sync again): %w", dataDir, err)
		}
		ahead, _, _, err = internal.GitAheadBehind(dataDir)
		if err != nil {
			return err
		}
	}

	if hasUpstream && ahead == 0 {
		fmt.Println("Already in sync with the remote")
		return nil
	}

	if err := internal.GitPush(dataDir); err != nil {
		return err
	}
	if hasUpstream {
		fmt.Printf("Pushed %s to the remote\n", internal.Bold(commits(ahead)))
	} else {
		fmt.Println("Pushed local entries to the remote")
	}
	return nil
}

func commits(n int) string {
	if n == 1 {
		return "1 commit"
	}
	return fmt.Sprintf("%d commits", n)
}
