package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitiseBranch(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"main", "main"},
		{"feature/auth", "feature+auth"},
		{"feature/nested/deep", "feature+nested+deep"},
	}
	for _, tt := range tests {
		got := SanitiseBranch(tt.input)
		if got != tt.want {
			t.Errorf("SanitiseBranch(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestJSONLPath(t *testing.T) {
	got := JSONLPath("/data", "my-project", "feature/auth")
	want := filepath.Join("/data", "logs", "my-project", "feature+auth.jsonl")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCollectPaths_AllProjects(t *testing.T) {
	dir := t.TempDir()
	// Create two projects with JSONL files
	os.MkdirAll(filepath.Join(dir, "logs", "alpha"), 0o755)
	os.MkdirAll(filepath.Join(dir, "logs", "beta"), 0o755)
	os.WriteFile(filepath.Join(dir, "logs", "alpha", "main.jsonl"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "logs", "beta", "main.jsonl"), []byte("{}"), 0o644)

	paths := CollectPaths(dir, "", "")
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
}

func TestCollectPaths_ScopedToProject(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "logs", "alpha"), 0o755)
	os.MkdirAll(filepath.Join(dir, "logs", "beta"), 0o755)
	os.WriteFile(filepath.Join(dir, "logs", "alpha", "main.jsonl"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "logs", "alpha", "dev.jsonl"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "logs", "beta", "main.jsonl"), []byte("{}"), 0o644)

	paths := CollectPaths(dir, "alpha", "")
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
}

func TestCollectPaths_ScopedToProjectAndBranch(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "logs", "alpha"), 0o755)
	os.WriteFile(filepath.Join(dir, "logs", "alpha", "main.jsonl"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "logs", "alpha", "dev.jsonl"), []byte("{}"), 0o644)

	paths := CollectPaths(dir, "alpha", "main")
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
}

func TestCollectPaths_NoLogsDir(t *testing.T) {
	paths := CollectPaths(t.TempDir(), "", "")
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}
