package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func seedDataRepo(t *testing.T) string {
	t.Helper()
	dir := initRepo(t)
	mustGit(t, dir, "commit", "--allow-empty", "-m", "init")
	return dir
}

func seedLog(t *testing.T, dataDir, project, branch, id string) {
	t.Helper()
	logs := filepath.Join(dataDir, "logs", project)
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	base := SanitiseBranch(branch)
	if err := os.WriteFile(filepath.Join(logs, base+".jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if id != "" {
		if err := WriteSidecar(filepath.Join(logs, base+".jsonl.id"), id); err != nil {
			t.Fatal(err)
		}
	}
	mustGit(t, dataDir, "add", "-A")
	mustGit(t, dataDir, "commit", "-m", "seed")
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s", args, out)
	}
	return strings.TrimSpace(string(out))
}

func TestMoveBranchLogs_RenamesFilesAndSidecarAndCommits(t *testing.T) {
	data := seedDataRepo(t)
	seedLog(t, data, "proj", "old", "id1")

	if err := MoveBranchLogs(data, "proj", "old", "new"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(data, "logs", "proj", "old.jsonl")); !os.IsNotExist(err) {
		t.Error("old.jsonl still present")
	}
	if _, err := os.Stat(filepath.Join(data, "logs", "proj", "new.jsonl")); err != nil {
		t.Errorf("new.jsonl missing: %v", err)
	}
	if got, _ := ReadSidecar(filepath.Join(data, "logs", "proj", "new.jsonl.id")); got != "id1" {
		t.Errorf("sidecar id = %q, want id1", got)
	}
	if status := gitOutput(t, data, "status", "--porcelain"); status != "" {
		t.Errorf("repo not clean after move: %q", status)
	}
}

func TestMoveBranchLogs_RefusesCollision(t *testing.T) {
	data := seedDataRepo(t)
	seedLog(t, data, "proj", "old", "id1")
	seedLog(t, data, "proj", "new", "id2")

	if err := MoveBranchLogs(data, "proj", "old", "new"); err == nil {
		t.Fatal("expected refusal when destination logfile exists")
	}
}

func TestReconcileBranch_StampsFreshAnchorOnFirstSight(t *testing.T) {
	cwd := seedDataRepo(t)
	data := t.TempDir()

	path, warn := ReconcileBranch(data, cwd, "proj", "main")
	if warn != nil {
		t.Fatal(warn)
	}
	if want := filepath.Join(data, "logs", "proj", "main.jsonl"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	id, _ := GitConfigGet(cwd, AnchorConfigKey("main"))
	if len(id) != 32 {
		t.Fatalf("anchor not stamped, got %q", id)
	}
	if sc, _ := ReadSidecar(filepath.Join(data, "logs", "proj", "main.jsonl.id")); sc != id {
		t.Errorf("sidecar %q != anchor %q", sc, id)
	}
}

func TestReconcileBranch_FollowsRename(t *testing.T) {
	cwd := seedDataRepo(t)
	data := seedDataRepo(t)
	seedLog(t, data, "proj", "old", "anchored-id")
	mustGit(t, cwd, "config", AnchorConfigKey("new"), "anchored-id")

	path, warn := ReconcileBranch(data, cwd, "proj", "new")
	if warn != nil {
		t.Fatal(warn)
	}
	if want := filepath.Join(data, "logs", "proj", "new.jsonl"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if _, err := os.Stat(filepath.Join(data, "logs", "proj", "old.jsonl")); !os.IsNotExist(err) {
		t.Error("old.jsonl still present after rename follow")
	}
	if _, err := os.Stat(filepath.Join(data, "logs", "proj", "new.jsonl")); err != nil {
		t.Errorf("new.jsonl missing: %v", err)
	}
}

func TestReconcileBranch_NoopWhenAlreadyCurrent(t *testing.T) {
	cwd := seedDataRepo(t)
	data := seedDataRepo(t)
	seedLog(t, data, "proj", "main", "the-id")
	mustGit(t, cwd, "config", AnchorConfigKey("main"), "the-id")

	path, warn := ReconcileBranch(data, cwd, "proj", "main")
	if warn != nil {
		t.Fatal(warn)
	}
	if want := filepath.Join(data, "logs", "proj", "main.jsonl"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if status := gitOutput(t, data, "status", "--porcelain"); status != "" {
		t.Errorf("unexpected repo change on no-op: %q", status)
	}
}

func TestNewID_Is32HexCharsAndUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewID()
		if len(id) != 32 {
			t.Fatalf("id %q has length %d, want 32", id, len(id))
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}

func TestAnchorConfigKey_UsesRawBranchName(t *testing.T) {
	if got := AnchorConfigKey("feature/foo"); got != "branch.feature/foo.jotter-id" {
		t.Errorf("got %q, want %q", got, "branch.feature/foo.jotter-id")
	}
}

func TestSidecarPath_SanitisesBranch(t *testing.T) {
	got, err := SidecarPath("/data", "proj", "feature/foo")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/data", "logs", "proj", "feature+foo.jsonl.id")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSidecar_RoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.jsonl.id")

	if got, err := ReadSidecar(path); err != nil || got != "" {
		t.Fatalf("absent sidecar: got %q err %v, want empty", got, err)
	}
	if err := WriteSidecar(path, "abc123"); err != nil {
		t.Fatal(err)
	}
	got, err := ReadSidecar(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "abc123" {
		t.Errorf("got %q, want %q", got, "abc123")
	}
}

func TestFindLogByID(t *testing.T) {
	dataDir := t.TempDir()
	logs := filepath.Join(dataDir, "logs", "proj")
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteSidecar(filepath.Join(logs, "main.jsonl.id"), "id-main"); err != nil {
		t.Fatal(err)
	}
	if err := WriteSidecar(filepath.Join(logs, "feature+foo.jsonl.id"), "id-foo"); err != nil {
		t.Fatal(err)
	}

	got, err := FindLogByID(dataDir, "proj", "id-foo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "feature+foo" {
		t.Errorf("got %q, want sanitised basename %q", got, "feature+foo")
	}

	missing, err := FindLogByID(dataDir, "proj", "nope")
	if err != nil {
		t.Fatal(err)
	}
	if missing != "" {
		t.Errorf("got %q, want empty for unknown id", missing)
	}
}
