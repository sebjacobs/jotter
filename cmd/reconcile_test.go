package cmd_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initProjectRepo creates a temp git repo on the given branch with one commit,
// standing in for a user's project checkout (cwd) during a write.
func initProjectRepo(t *testing.T, branch string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"git", "init", "-b", branch},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	} {
		gitRun(t, dir, args[1:]...)
	}
	return dir
}

// runJotterInRepo runs jotter with cwd inside a project repo and a .jotter file
// there pointing at dataDir — the combination the reconcile path needs that
// neither runJotter nor runJotterFromGitRepo provides alone.
func runJotterInRepo(t *testing.T, repoDir, dataDir string, args ...string) (string, string, int) {
	t.Helper()
	body := fmt.Sprintf("data_dir = %q\n", dataDir)
	if err := os.WriteFile(filepath.Join(repoDir, ".jotter"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("run jotter: %v", err)
		}
	}
	return stdout.String(), stderr.String(), code
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s", args, out)
	}
}

func gitCapture(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s", args, out)
	}
	return strings.TrimSpace(string(out))
}

// configValue reads a git config key, returning "" when unset.
func configValue(t *testing.T, dir, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func globOne(t *testing.T, pattern string) string {
	t.Helper()
	matches, _ := filepath.Glob(pattern)
	if len(matches) != 1 {
		t.Fatalf("want exactly one match for %s, got %v", pattern, matches)
	}
	return matches[0]
}

func TestWrite_StampsAnchorAndSidecarOnFreshBranch(t *testing.T) {
	data := initDataDir(t)
	repo := initProjectRepo(t, "main")

	_, stderr, code := runJotterInRepo(t, repo, data, "write", "--type", "note", "--content", "hi")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}

	globOne(t, filepath.Join(data, "logs", "*", "main.jsonl"))
	globOne(t, filepath.Join(data, "logs", "*", "main.jsonl.id"))
	if id := configValue(t, repo, "branch.main.jotter-id"); len(id) != 32 {
		t.Errorf("anchor not stamped in project repo, got %q", id)
	}
}

func TestWrite_FollowsBranchRename(t *testing.T) {
	data := initDataDir(t)
	repo := initProjectRepo(t, "old")

	runJotterInRepo(t, repo, data, "write", "--type", "note", "--content", "first")
	gitRun(t, repo, "branch", "-m", "old", "new")
	_, stderr, code := runJotterInRepo(t, repo, data, "write", "--type", "note", "--content", "second")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}

	if matches, _ := filepath.Glob(filepath.Join(data, "logs", "*", "old.jsonl")); len(matches) != 0 {
		t.Errorf("old.jsonl still present after rename: %v", matches)
	}
	newLog := globOne(t, filepath.Join(data, "logs", "*", "new.jsonl"))
	body, _ := os.ReadFile(newLog)
	if !strings.Contains(string(body), "first") || !strings.Contains(string(body), "second") {
		t.Errorf("history not continuous in new.jsonl:\n%s", body)
	}
	if log := gitCapture(t, data, "log", "--oneline"); !strings.Contains(log, "rename:") {
		t.Errorf("no rename commit in data repo:\n%s", log)
	}
}

func TestWrite_AdoptsPreexistingLogThenFollowsRename(t *testing.T) {
	data := initDataDir(t)
	repo := initProjectRepo(t, "main")
	projOut, _, _ := runJotterInRepo(t, repo, data, "project")
	project := strings.TrimSpace(projOut)

	logs := filepath.Join(data, "logs", project)
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logs, "main.jsonl"), []byte("{\"type\": \"note\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, data, "add", "-A")
	gitRun(t, data, "commit", "-m", "pre-existing log")

	_, stderr, code := runJotterInRepo(t, repo, data, "write", "--type", "note", "--content", "after")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if id := configValue(t, repo, "branch.main.jotter-id"); len(id) != 32 {
		t.Fatalf("pre-existing log not adopted (no anchor): %q", id)
	}
	globOne(t, filepath.Join(logs, "main.jsonl.id"))

	gitRun(t, repo, "branch", "-m", "main", "renamed")
	if _, stderr, code := runJotterInRepo(t, repo, data, "write", "--type", "note", "--content", "post-rename"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if matches, _ := filepath.Glob(filepath.Join(logs, "main.jsonl")); len(matches) != 0 {
		t.Errorf("main.jsonl still present after adopt+rename: %v", matches)
	}
	globOne(t, filepath.Join(logs, "renamed.jsonl"))
}

func TestWrite_OffBranchSkipsTracking(t *testing.T) {
	data := initDataDir(t)
	repo := initProjectRepo(t, "main")

	_, stderr, code := runJotterInRepo(t, repo, data, "write", "--branch", "elsewhere", "--type", "note", "--content", "hi")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if id := configValue(t, repo, "branch.elsewhere.jotter-id"); id != "" {
		t.Errorf("anchor written for off-branch write: %q", id)
	}
	if matches, _ := filepath.Glob(filepath.Join(data, "logs", "*", "*.jsonl.id")); len(matches) != 0 {
		t.Errorf("sidecar created on off-branch write: %v", matches)
	}
}

func TestSidecarInvisibleToLsAndTail(t *testing.T) {
	data := initDataDir(t)
	repo := initProjectRepo(t, "main")
	runJotterInRepo(t, repo, data, "write", "--type", "note", "--content", "hi")

	lsOut, _, _ := runJotterInRepo(t, repo, data, "ls")
	if strings.Contains(lsOut, ".id") {
		t.Errorf("ls leaked sidecar:\n%s", lsOut)
	}
	tailOut, stderr, code := runJotterInRepo(t, repo, data, "tail")
	if code != 0 {
		t.Fatalf("tail exit %d: %s", code, stderr)
	}
	if strings.Contains(tailOut, ".id") {
		t.Errorf("tail leaked sidecar:\n%s", tailOut)
	}
}

func TestBranchMv_MovesLogsAndSidecar(t *testing.T) {
	data := initDataDir(t)
	repo := initProjectRepo(t, "old")
	runJotterInRepo(t, repo, data, "write", "--type", "note", "--content", "hi")

	stdout, stderr, code := runJotterInRepo(t, repo, data, "branch", "mv", "old", "new")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Renamed branch logs") {
		t.Errorf("unexpected stdout: %s", stdout)
	}
	if matches, _ := filepath.Glob(filepath.Join(data, "logs", "*", "old.jsonl")); len(matches) != 0 {
		t.Errorf("old.jsonl still present: %v", matches)
	}
	globOne(t, filepath.Join(data, "logs", "*", "new.jsonl"))
	globOne(t, filepath.Join(data, "logs", "*", "new.jsonl.id"))
}

func TestBranchMv_RefusesCollision(t *testing.T) {
	data := initDataDir(t)
	repo := initProjectRepo(t, "old")
	runJotterInRepo(t, repo, data, "write", "--type", "note", "--content", "a")
	runJotterInRepo(t, repo, data, "write", "--branch", "new", "--type", "note", "--content", "b")

	_, _, code := runJotterInRepo(t, repo, data, "branch", "mv", "old", "new")
	if code == 0 {
		t.Fatal("expected non-zero exit when destination logs exist")
	}
}

func TestBranch_BareStillPrintsCurrentBranch(t *testing.T) {
	data := initDataDir(t)
	repo := initProjectRepo(t, "feature/x")

	stdout, stderr, code := runJotterInRepo(t, repo, data, "branch")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if strings.TrimSpace(stdout) != "feature/x" {
		t.Errorf("branch = %q, want feature/x", strings.TrimSpace(stdout))
	}
}
