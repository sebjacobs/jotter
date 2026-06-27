package internal

import (
	"os/exec"
	"testing"
)

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	return dir
}

func TestGitConfigGet_UnsetKeyReturnsEmpty(t *testing.T) {
	dir := initRepo(t)
	got, err := GitConfigGet(dir, "branch.main.jotter-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string for unset key", got)
	}
}

func TestGitConfigSetThenGet_RoundTrips(t *testing.T) {
	dir := initRepo(t)
	if err := GitConfigSet(dir, "branch.main.jotter-id", "abc123"); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	got, err := GitConfigGet(dir, "branch.main.jotter-id")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got != "abc123" {
		t.Errorf("got %q, want %q", got, "abc123")
	}
}

func TestGitConfigGet_SubsectionWithSlash(t *testing.T) {
	dir := initRepo(t)
	if err := GitConfigSet(dir, "branch.feature/foo.jotter-id", "deadbeef"); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	got, err := GitConfigGet(dir, "branch.feature/foo.jotter-id")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got != "deadbeef" {
		t.Errorf("got %q, want %q", got, "deadbeef")
	}
}

func TestGitConfigSet_SurvivesBranchRename(t *testing.T) {
	dir := initRepo(t)
	mustGit(t, dir, "commit", "--allow-empty", "-m", "init")
	if err := GitConfigSet(dir, "branch.main.jotter-id", "stableid"); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	mustGit(t, dir, "branch", "-m", "main", "renamed")

	got, err := GitConfigGet(dir, "branch.renamed.jotter-id")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got != "stableid" {
		t.Errorf("after rename got %q, want %q — config section did not travel", got, "stableid")
	}
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s", args, out)
	}
}
