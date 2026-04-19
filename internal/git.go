package internal

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitProjectName returns the basename of the git toplevel for cwd.
// Returns an error if cwd is not inside a git repo.
func GitProjectName(cwd string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repo (run from inside one, or pass --project explicitly)")
	}
	return filepath.Base(strings.TrimSpace(string(out))), nil
}

// GitCurrentBranch returns the current branch name for cwd.
// Returns an error if cwd is not inside a git repo, or if HEAD is detached.
func GitCurrentBranch(cwd string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repo (run from inside one, or pass --branch explicitly)")
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return "", fmt.Errorf("detached HEAD — pass --branch explicitly")
	}
	return branch, nil
}

// GitCommit stages a file and commits it in the data repo.
func GitCommit(dataDir, filePath, message string) error {
	if err := run(dataDir, "git", "add", filePath); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	if err := run(dataDir, "git", "commit", "-m", message); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

// GitPush pushes the data repo to its remote.
func GitPush(dataDir string) error {
	if err := run(dataDir, "git", "push"); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

// GitHasRemote reports whether the data repo has any remote configured.
func GitHasRemote(dataDir string) bool {
	cmd := exec.Command("git", "remote")
	cmd.Dir = dataDir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(bytes.TrimSpace(out)) > 0
}

func run(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, out)
	}
	return nil
}
