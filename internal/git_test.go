package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"git", "init", "-b", "main"},
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

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// appendCommit appends a line to relPath (relative to dir) and commits it,
// returning the abbreviated SHA of the resulting commit. It models jotter's
// one-entry-per-commit append.
func appendCommit(t *testing.T, dir, relPath, line string) string {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(full, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(line + "\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	mustGit(t, dir, "add", relPath)
	mustGit(t, dir, "commit", "-m", "append")
	return gitOut(t, dir, "rev-parse", "HEAD")[:shortSHALen]
}

func TestGitHeadSHA_MatchesRevParse(t *testing.T) {
	dir := initRepo(t)
	mustGit(t, dir, "commit", "--allow-empty", "-m", "one")

	got, err := GitHeadSHA(dir)
	if err != nil {
		t.Fatalf("GitHeadSHA: %v", err)
	}
	want := gitOut(t, dir, "rev-parse", "HEAD")[:shortSHALen]
	if got != want {
		t.Errorf("GitHeadSHA = %q, want %q", got, want)
	}
	if len(got) != shortSHALen {
		t.Errorf("expected %d-char abbreviation, got %q", shortSHALen, got)
	}
}

func TestGitLineSHAs_MapsEachLineToItsCommit(t *testing.T) {
	dir := initRepo(t)
	rel := "logs/proj/main.jsonl"
	want := []string{
		appendCommit(t, dir, rel, `{"content":"one"}`),
		appendCommit(t, dir, rel, `{"content":"two"}`),
		appendCommit(t, dir, rel, `{"content":"three"}`),
	}

	got, err := GitLineSHAs(dir, rel)
	if err != nil {
		t.Fatalf("GitLineSHAs: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d shas, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d sha = %q, want %q (the commit that appended it)", i, got[i], want[i])
		}
	}
}

func TestGitLineSHAs_FollowsRename(t *testing.T) {
	dir := initRepo(t)
	rel := "logs/proj/main.jsonl"
	first := appendCommit(t, dir, rel, `{"content":"one"}`)

	renamed := "logs/proj/renamed.jsonl"
	mustGit(t, dir, "mv", rel, renamed)
	mustGit(t, dir, "commit", "-m", "rename branch log")
	second := appendCommit(t, dir, renamed, `{"content":"two"}`)

	got, err := GitLineSHAs(dir, renamed)
	if err != nil {
		t.Fatalf("GitLineSHAs: %v", err)
	}
	want := []string{first, second}
	if len(got) != len(want) {
		t.Fatalf("got %d shas, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d sha = %q, want %q — blame did not follow the rename", i, got[i], want[i])
		}
	}
}

func TestGitLineSHAs_SkipsBlankLines(t *testing.T) {
	dir := initRepo(t)
	rel := "logs/proj/main.jsonl"
	one := appendCommit(t, dir, rel, `{"content":"one"}`)
	appendCommit(t, dir, rel, "")
	two := appendCommit(t, dir, rel, `{"content":"two"}`)

	got, err := GitLineSHAs(dir, rel)
	if err != nil {
		t.Fatalf("GitLineSHAs: %v", err)
	}
	if want := []string{one, two}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v — blank line should be skipped to stay aligned with ReadEntries", got, want)
	}
}

func TestGitLineSHAs_IgnoresConfiguredIgnoreRevsFile(t *testing.T) {
	dir := initRepo(t)
	rel := "logs/proj/main.jsonl"
	one := appendCommit(t, dir, rel, `{"content":"one"}`)
	mustGit(t, dir, "config", "blame.ignoreRevsFile", ".git-blame-ignore-revs")

	got, err := GitLineSHAs(dir, rel)
	if err != nil {
		t.Fatalf("GitLineSHAs: %v", err)
	}
	if len(got) != 1 || got[0] != one {
		t.Errorf("got %v, want [%s]", got, one)
	}
}
