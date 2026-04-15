package internal

import (
	"fmt"
	"os/exec"
)

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

func run(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, out)
	}
	return nil
}
