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

// configWorkdir returns a fresh directory holding a .jotter pointing at store,
// suitable for passing as `mv --from` so jotter resolves that store from it.
func configWorkdir(t *testing.T, store string) string {
	t.Helper()
	dir := t.TempDir()
	body := []byte("data_dir = \"" + store + "\"\n")
	if err := os.WriteFile(filepath.Join(dir, ".jotter"), body, 0o644); err != nil {
		t.Fatalf("writing .jotter: %v", err)
	}
	return dir
}

func TestMv_CrossStore_RelocatesKeepingName(t *testing.T) {
	src := initDataDir(t)
	dest := initDataDir(t)
	seedEntry(t, src, "proj", "main")

	stdout, stderr, code := runJotter(t, dest, "mv", "proj", "proj", "--from", configWorkdir(t, src))
	if code != 0 {
		t.Fatalf("exit %d: %s %s", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(src, "logs", "proj")); !os.IsNotExist(err) {
		t.Errorf("source logs dir still present")
	}
	if _, err := os.Stat(filepath.Join(dest, "logs", "proj", "main.jsonl")); err != nil {
		t.Errorf("destination logs file missing: %v", err)
	}
	if !strings.Contains(stdout, "Relocated project") {
		t.Errorf("unexpected stdout: %s", stdout)
	}
}

func TestMv_CrossStore_CommitsBothStoresLeavingThemClean(t *testing.T) {
	src := initDataDir(t)
	dest := initDataDir(t)
	seedEntry(t, src, "proj", "main")

	if _, _, code := runJotter(t, dest, "mv", "proj", "proj", "--from", configWorkdir(t, src)); code != 0 {
		t.Fatalf("mv exit %d", code)
	}

	if status := git(t, src, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("source repo dirty after relocate: %q", status)
	}
	if status := git(t, dest, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("destination repo dirty after relocate: %q", status)
	}
	if subject := git(t, src, "log", "-1", "--format=%s"); !strings.Contains(subject, "relocate: move logs/proj") {
		t.Errorf("unexpected source commit subject: %q", subject)
	}
	if subject := git(t, dest, "log", "-1", "--format=%s"); !strings.Contains(subject, "relocate: bring in logs/proj") {
		t.Errorf("unexpected destination commit subject: %q", subject)
	}
}

func TestMv_CrossStore_CanRenameWhileRelocating(t *testing.T) {
	src := initDataDir(t)
	dest := initDataDir(t)
	seedEntry(t, src, "old-proj", "feature/x")

	if _, stderr, code := runJotter(t, dest, "mv", "old-proj", "new-proj", "--from", configWorkdir(t, src)); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}

	data, err := os.ReadFile(filepath.Join(dest, "logs", "new-proj", "feature+x.jsonl"))
	if err != nil {
		t.Fatalf("relocated branch file missing: %v", err)
	}
	if !strings.Contains(string(data), "seed") {
		t.Errorf("entry content lost: %s", data)
	}
}

func TestMv_CrossStore_ErrorsWhenSourceMissing(t *testing.T) {
	src := initDataDir(t)
	dest := initDataDir(t)

	_, stderr, code := runJotter(t, dest, "mv", "nope", "nope", "--from", configWorkdir(t, src))
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "no logs for project") {
		t.Errorf("stderr: %s", stderr)
	}
}

func TestMv_CrossStore_ErrorsWhenDestinationExists(t *testing.T) {
	src := initDataDir(t)
	dest := initDataDir(t)
	seedEntry(t, src, "proj", "main")
	seedEntry(t, dest, "proj", "main")

	_, stderr, code := runJotter(t, dest, "mv", "proj", "proj", "--from", configWorkdir(t, src))
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "already has logs") {
		t.Errorf("stderr: %s", stderr)
	}
}

func TestMv_FromResolvingToSameStoreStillRejectsSameName(t *testing.T) {
	dir := initDataDir(t)
	seedEntry(t, dir, "same", "main")

	_, stderr, code := runJotter(t, dir, "mv", "same", "same", "--from", configWorkdir(t, dir))
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "same") {
		t.Errorf("stderr: %s", stderr)
	}
}
