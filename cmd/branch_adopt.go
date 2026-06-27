package cmd

import (
	"fmt"
	"os"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var branchAdoptCmd = &cobra.Command{
	Use:   "adopt",
	Short: "Anchor existing branches so future renames are tracked",
	Long: "Stamp a stable id on every local branch that already has logs but is not\n" +
		"yet anchored, so a later `git branch -m` is followed automatically.\n\n" +
		"Run once per repo to migrate pre-existing history into branch tracking —\n" +
		"a write would otherwise only anchor a branch the next time you write to it,\n" +
		"leaving a branch renamed before then unprotected. Idempotent: already-anchored\n" +
		"branches and branches without logs are skipped. --project defaults to the\n" +
		"basename of the git toplevel.",
	Args: cobra.NoArgs,
	RunE: runBranchAdopt,
}

func init() {
	branchAdoptCmd.Flags().String("project", "", "Project name (default: basename of the git toplevel)")
	_ = branchAdoptCmd.RegisterFlagCompletionFunc("project", completeProjects)
	branchCmd.AddCommand(branchAdoptCmd)
}

func runBranchAdopt(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	project, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	dataDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}
	branches, err := internal.GitLocalBranches(cwd)
	if err != nil {
		return err
	}

	adopted := 0
	for _, branch := range branches {
		logPath, err := internal.JSONLPath(dataDir, project, branch)
		if err != nil {
			return err
		}
		if _, err := os.Stat(logPath); err != nil {
			continue
		}
		if id, _ := internal.GitConfigGet(cwd, internal.AnchorConfigKey(branch)); id != "" {
			continue
		}
		if _, err := internal.AnchorBranch(dataDir, cwd, project, branch); err != nil {
			return err
		}
		sidecar, err := internal.SidecarPath(dataDir, project, branch)
		if err != nil {
			return err
		}
		if err := internal.GitAdd(dataDir, sidecar); err != nil {
			return err
		}
		adopted++
	}

	if adopted > 0 {
		if err := internal.GitCommitStaged(dataDir, fmt.Sprintf("chore: adopt %s branch ids", project)); err != nil {
			return err
		}
	}

	noun := "branches"
	if adopted == 1 {
		noun = "branch"
	}
	fmt.Printf("Adopted %d %s\n", adopted, noun)
	return nil
}
