package internal

import (
	"os"
	"path/filepath"
	"testing"
)

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
