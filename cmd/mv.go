package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var mvFromDir string

var mvCmd = &cobra.Command{
	Use:   "mv <old-project> <new-project>",
	Short: "Rename a project's logs, or move them to another store",
	Long: "Rename a project's log directory in the data repo and commit the move.\n\n" +
		"A project's jotter name is the basename of its git toplevel, so renaming the\n" +
		"project directory on disk orphans its logs under the old name. Run this to\n" +
		"carry them over: logs/<old-project> is renamed to logs/<new-project> and the\n" +
		"rename is committed (locally — like every entry, the push to the remote is left to the background timer).\n\n" +
		"Moving a project to a directory served by a different .jotter store (e.g. between\n" +
		"a client tree and a personal tree) relocates its logs across stores: pass --from\n" +
		"<old-dir> so the source store resolves from where the project used to live while\n" +
		"the destination store resolves from the current directory. A cross-store move may\n" +
		"keep the same name (logs/<name> in one store becomes logs/<name> in the other);\n" +
		"the removal is committed in the source store and the addition in the destination.",
	Args:              cobra.ExactArgs(2),
	RunE:              runMv,
	ValidArgsFunction: completeMvArgs,
}

func init() {
	mvCmd.Flags().StringVar(&mvFromDir, "from", "",
		"directory whose .jotter store holds the source logs; enables a cross-store move")
	rootCmd.AddCommand(mvCmd)
}

func runMv(_ *cobra.Command, args []string) error {
	oldProject, newProject := args[0], args[1]

	if err := internal.ValidatePathComponent("project", oldProject); err != nil {
		return err
	}
	if err := internal.ValidatePathComponent("project", newProject); err != nil {
		return err
	}

	destDir, err := internal.GetDataDir()
	if err != nil {
		return err
	}

	srcDir := destDir
	if mvFromDir != "" {
		srcDir, err = internal.GetDataDirFrom(mvFromDir)
		if err != nil {
			return fmt.Errorf("resolving --from store: %w", err)
		}
	}

	if filepath.Clean(srcDir) == filepath.Clean(destDir) {
		return renameInStore(destDir, oldProject, newProject)
	}
	return relocateAcrossStores(srcDir, destDir, oldProject, newProject)
}

// renameInStore renames logs/<old> to logs/<new> within a single store and
// commits the rename. The two names must differ — a same-name rename within one
// store is a no-op the caller almost certainly didn't intend.
func renameInStore(dataDir, oldProject, newProject string) error {
	if oldProject == newProject {
		return fmt.Errorf("old and new project names are the same: %q", oldProject)
	}

	oldRel := filepath.Join("logs", oldProject)
	newRel := filepath.Join("logs", newProject)

	if info, err := os.Stat(filepath.Join(dataDir, oldRel)); err != nil || !info.IsDir() {
		return fmt.Errorf("no logs for project %q", oldProject)
	}
	if _, err := os.Stat(filepath.Join(dataDir, newRel)); err == nil {
		return fmt.Errorf("project %q already has logs — refusing to overwrite", newProject)
	}

	if err := internal.GitMove(dataDir, oldRel, newRel); err != nil {
		return fmt.Errorf("renaming logs: %w", err)
	}
	if err := internal.GitCommitStaged(dataDir, fmt.Sprintf("rename: %s -> %s", oldRel, newRel)); err != nil {
		return fmt.Errorf("committing rename: %w", err)
	}

	fmt.Printf("Renamed project %s -> %s\n", internal.Dim(oldProject), internal.Dim(newProject))
	return nil
}

// relocateAcrossStores moves logs/<old> out of the source store and into the
// destination store as logs/<new>, committing the removal in the source and the
// addition in the destination. Unlike a same-store rename the two names may be
// equal — keeping a project's name while its directory changes category is the
// common case — so only a real destination collision is refused.
func relocateAcrossStores(srcDir, destDir, oldProject, newProject string) error {
	oldRel := filepath.Join("logs", oldProject)
	newRel := filepath.Join("logs", newProject)
	srcPath := filepath.Join(srcDir, oldRel)
	destPath := filepath.Join(destDir, newRel)

	if info, err := os.Stat(srcPath); err != nil || !info.IsDir() {
		return fmt.Errorf("no logs for project %q", oldProject)
	}
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("project %q already has logs in the destination store — refusing to overwrite", newProject)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("preparing destination: %w", err)
	}
	if err := os.Rename(srcPath, destPath); err != nil {
		return fmt.Errorf("moving logs across stores: %w", err)
	}

	if err := internal.GitCommit(srcDir, oldRel, fmt.Sprintf("relocate: move %s to %s", oldRel, destDir)); err != nil {
		return fmt.Errorf("committing removal in source store: %w", err)
	}
	if err := internal.GitCommit(destDir, newRel, fmt.Sprintf("relocate: bring in %s from %s", newRel, srcDir)); err != nil {
		return fmt.Errorf("committing addition in destination store: %w", err)
	}

	fmt.Printf("Relocated project %s -> %s across stores\n", internal.Dim(oldProject), internal.Dim(newProject))
	return nil
}

// completeMvArgs completes the old-project positional with known project names;
// the new-project positional is free text, so it offers nothing.
func completeMvArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completeProjects(cmd, args, toComplete)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
