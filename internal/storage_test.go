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
	got, err := JSONLPath("/data", "my-project", "feature/auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("/data", "logs", "my-project", "feature+auth.jsonl")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestJSONLPath_Rejects(t *testing.T) {
	cases := []struct{ project, branch string }{
		{"..", "main"},
		{"../etc", "main"},
		{"a/b", "main"},
		{"", "main"},
		{".hidden", "main"},
		{"proj", ".."},
		{"proj", ""},
		{"proj", ".hidden"},
		{"proj\x00null", "main"},
	}
	for _, tc := range cases {
		if _, err := JSONLPath("/data", tc.project, tc.branch); err == nil {
			t.Errorf("JSONLPath(%q, %q) accepted unsafe input", tc.project, tc.branch)
		}
	}
}

func TestValidatePathComponent_AcceptsNormal(t *testing.T) {
	ok := []string{"main", "feature+auth", "my-project", "proj_123", "a.b"}
	for _, v := range ok {
		if err := ValidatePathComponent("test", v); err != nil {
			t.Errorf("ValidatePathComponent(%q) rejected valid input: %v", v, err)
		}
	}
}

func TestCollectPaths_AllProjects(t *testing.T) {
	dir := t.TempDir()
	// Create two projects with JSONL files
	os.MkdirAll(filepath.Join(dir, "logs", "alpha"), 0o755)
	os.MkdirAll(filepath.Join(dir, "logs", "beta"), 0o755)
	os.WriteFile(filepath.Join(dir, "logs", "alpha", "main.jsonl"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "logs", "beta", "main.jsonl"), []byte("{}"), 0o644)

	paths, _ := CollectPaths(dir, "", "")
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

	paths, _ := CollectPaths(dir, "alpha", "")
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
}

func TestCollectPaths_ScopedToProjectAndBranch(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "logs", "alpha"), 0o755)
	os.WriteFile(filepath.Join(dir, "logs", "alpha", "main.jsonl"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "logs", "alpha", "dev.jsonl"), []byte("{}"), 0o644)

	paths, _ := CollectPaths(dir, "alpha", "main")
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
}

func TestCollectPaths_NoLogsDir(t *testing.T) {
	paths, _ := CollectPaths(t.TempDir(), "", "")
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}
