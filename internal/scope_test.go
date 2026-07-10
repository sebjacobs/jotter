package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func seedLogs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "logs", "alpha"), 0o755)
	os.MkdirAll(filepath.Join(dir, "logs", "beta"), 0o755)
	os.WriteFile(filepath.Join(dir, "logs", "alpha", "main.jsonl"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "logs", "alpha", "dev.jsonl"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "logs", "beta", "main.jsonl"), []byte("{}"), 0o644)
	return dir
}

func TestAllScope_Paths(t *testing.T) {
	paths, err := AllScope{}.Paths(seedLogs(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}
}

func TestProjectScope_Paths(t *testing.T) {
	paths, err := ProjectScope{Project: "alpha"}.Paths(seedLogs(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
}

func TestBranchScope_Paths(t *testing.T) {
	paths, err := BranchScope{Project: "alpha", Branch: "main"}.Paths(seedLogs(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
}

func TestAllScope_NoLogsDir(t *testing.T) {
	paths, err := AllScope{}.Paths(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}

func TestScope_RejectsUnsafeComponents(t *testing.T) {
	dir := t.TempDir()
	if _, err := (ProjectScope{Project: ".."}).Paths(dir); err == nil {
		t.Error("ProjectScope accepted unsafe project")
	}
	if _, err := (BranchScope{Project: "proj", Branch: ".."}).Paths(dir); err == nil {
		t.Error("BranchScope accepted unsafe branch")
	}
}
