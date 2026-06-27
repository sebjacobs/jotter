package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedEntry writes and commits one entry so a project's logs exist and are tracked.
func seedEntry(t *testing.T, dataDir, project, branch string) {
	t.Helper()
	_, stderr, code := runJotter(t, dataDir,
		"write", "--project", project, "--branch", branch,
		"--type", "start", "--content", "seed")
	if code != 0 {
		t.Fatalf("seed write failed (%d): %s", code, stderr)
	}
}

func TestMv_RenamesProjectLogs(t *testing.T) {
	dir := initDataDir(t)
	seedEntry(t, dir, "old-proj", "main")

	stdout, stderr, code := runJotter(t, dir, "mv", "old-proj", "new-proj")
	if code != 0 {
		t.Fatalf("exit %d: %s %s", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "logs", "old-proj")); !os.IsNotExist(err) {
		t.Errorf("old logs dir still present")
	}
	if _, err := os.Stat(filepath.Join(dir, "logs", "new-proj", "main.jsonl")); err != nil {
		t.Errorf("new logs file missing: %v", err)
	}
	if !strings.Contains(stdout, "Renamed project") {
		t.Errorf("unexpected stdout: %s", stdout)
	}
}

func TestMv_CommitsRenameLeavingRepoClean(t *testing.T) {
	dir := initDataDir(t)
	seedEntry(t, dir, "old-proj", "main")

	if _, _, code := runJotter(t, dir, "mv", "old-proj", "new-proj"); code != 0 {
		t.Fatalf("mv exit %d", code)
	}

	if status := git(t, dir, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("data repo dirty after mv: %q", status)
	}
	if subject := git(t, dir, "log", "-1", "--format=%s"); !strings.Contains(subject, "rename: logs/old-proj -> logs/new-proj") {
		t.Errorf("unexpected commit subject: %q", subject)
	}
}

func TestMv_PreservesEntryContentAcrossRename(t *testing.T) {
	dir := initDataDir(t)
	seedEntry(t, dir, "old-proj", "feature/x")

	runJotter(t, dir, "mv", "old-proj", "new-proj")

	data, err := os.ReadFile(filepath.Join(dir, "logs", "new-proj", "feature+x.jsonl"))
	if err != nil {
		t.Fatalf("renamed branch file missing: %v", err)
	}
	if !strings.Contains(string(data), "seed") {
		t.Errorf("entry content lost: %s", data)
	}
}

func TestMv_ErrorsWhenSourceMissing(t *testing.T) {
	dir := initDataDir(t)
	_, stderr, code := runJotter(t, dir, "mv", "nope", "new-proj")
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "no logs for project") {
		t.Errorf("stderr: %s", stderr)
	}
}

func TestMv_ErrorsWhenDestinationExists(t *testing.T) {
	dir := initDataDir(t)
	seedEntry(t, dir, "alpha", "main")
	seedEntry(t, dir, "beta", "main")

	_, stderr, code := runJotter(t, dir, "mv", "alpha", "beta")
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "already has logs") {
		t.Errorf("stderr: %s", stderr)
	}
}

func TestMv_RejectsPathEscapingNames(t *testing.T) {
	dir := initDataDir(t)
	seedEntry(t, dir, "ok", "main")

	_, stderr, code := runJotter(t, dir, "mv", "ok", "../escape")
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "must not") {
		t.Errorf("stderr: %s", stderr)
	}
}

func TestMv_RejectsSameName(t *testing.T) {
	dir := initDataDir(t)
	seedEntry(t, dir, "same", "main")

	_, stderr, code := runJotter(t, dir, "mv", "same", "same")
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "same") {
		t.Errorf("stderr: %s", stderr)
	}
}
