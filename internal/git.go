package internal

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// GitProjectName returns the basename of the main repo's toplevel for cwd.
// Inside a worktree, this is the main repo's directory rather than the
// worktree's checkout dir — so logs stay grouped under the project name even
// when work happens in a worktree.
// Returns an error if cwd is not inside a git repo.
func GitProjectName(cwd string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repo (run from inside one, or pass --project explicitly)")
	}
	commonDir := strings.TrimSpace(string(out))
	return filepath.Base(filepath.Dir(commonDir)), nil
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

// GitConfigGet reads a git config value scoped to the repo at cwd. An unset key
// (where `git config --get` exits 1) returns "" with a nil error; any other
// failure returns the error.
func GitConfigGet(cwd, key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("git config --get %s: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GitConfigSet writes a git config value scoped to the repo at cwd.
func GitConfigSet(cwd, key, value string) error {
	return run(cwd, "git", "config", key, value)
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

// GitAdd stages a path within the data repo.
func GitAdd(dataDir, path string) error {
	return run(dataDir, "git", "add", path)
}

// GitLocalBranches returns the local branch names in cwd's repo.
func GitLocalBranches(cwd string) ([]string, error) {
	cmd := exec.Command("git", "for-each-ref", "--format=%(refname:short)", "refs/heads")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing branches: %w", err)
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// GitMove renames a tracked path within the data repo, staging the rename.
// from and to are relative to dataDir.
func GitMove(dataDir, from, to string) error {
	if err := run(dataDir, "git", "mv", from, to); err != nil {
		return fmt.Errorf("git mv: %w", err)
	}
	return nil
}

// GitCommitStaged commits whatever is already staged in the data repo. Pairs
// with GitMove, whose rename git has already added to the index.
func GitCommitStaged(dataDir, message string) error {
	if err := run(dataDir, "git", "commit", "-m", message); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

// GitPush pushes the data repo to its remote. On a branch that has no upstream
// configured yet (e.g. a freshly `jotter setup` repo whose first push never
// landed), it pushes with -u so the tracking branch is set and later pushes are
// bare.
func GitPush(dataDir string) error {
	if gitHasUpstream(dataDir) {
		if err := run(dataDir, "git", "push"); err != nil {
			return fmt.Errorf("git push: %w", err)
		}
		return nil
	}
	if err := run(dataDir, "git", "push", "-u", "origin", "HEAD"); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

// GitFetch updates the data repo's remote-tracking refs.
func GitFetch(dataDir string) error {
	if err := run(dataDir, "git", "fetch"); err != nil {
		return fmt.Errorf("git fetch: %w", err)
	}
	return nil
}

// GitPullRebase rebases the local branch onto its upstream, replaying local
// entries on top of any commits the remote gained since the last push.
func GitPullRebase(dataDir string) error {
	if err := run(dataDir, "git", "pull", "--rebase"); err != nil {
		return fmt.Errorf("git pull --rebase: %w", err)
	}
	return nil
}

// GitAheadBehind reports how many commits the local branch is ahead of and
// behind its upstream. hasUpstream is false when no tracking branch is
// configured, in which case ahead and behind are both zero.
func GitAheadBehind(dataDir string) (ahead, behind int, hasUpstream bool, err error) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", "@{u}...HEAD")
	cmd.Dir = dataDir
	out, runErr := cmd.Output()
	if runErr != nil {
		return 0, 0, false, nil
	}
	fields := strings.Fields(string(out))
	if len(fields) != 2 {
		return 0, 0, true, fmt.Errorf("unexpected rev-list output: %q", out)
	}
	behind, _ = strconv.Atoi(fields[0])
	ahead, _ = strconv.Atoi(fields[1])
	return ahead, behind, true, nil
}

// gitHasUpstream reports whether the current branch has a configured upstream.
func gitHasUpstream(dataDir string) bool {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "@{u}")
	cmd.Dir = dataDir
	return cmd.Run() == nil
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
