package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveEntryLine_DropsMatchingLineAndReportsRelPath(t *testing.T) {
	dir := t.TempDir()
	logfile := filepath.Join(dir, "logs", "proj", "main.jsonl")
	if err := os.MkdirAll(filepath.Dir(logfile), 0o755); err != nil {
		t.Fatal(err)
	}
	target := `{"timestamp": "2026-04-11T11:00:00", "type": "checkpoint", "content": "middle"}`
	first := `{"timestamp": "2026-04-11T10:00:00", "type": "start", "content": "first"}`
	last := `{"timestamp": "2026-04-11T12:00:00", "type": "note", "content": "last"}`
	if err := os.WriteFile(logfile, []byte(first+"\n"+target+"\n"+last+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rel, err := RemoveEntryLine(dir, target)
	if err != nil {
		t.Fatalf("RemoveEntryLine: %v", err)
	}
	if rel != filepath.Join("logs", "proj", "main.jsonl") {
		t.Errorf("rel = %q", rel)
	}
	entries, _ := ReadEntries(logfile)
	if len(entries) != 2 || entries[0].Content != "first" || entries[1].Content != "last" {
		t.Errorf("expected first+last to remain, got %+v", entries)
	}
}

func TestRemoveEntryLine_ErrorsWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	logfile := filepath.Join(dir, "logs", "proj", "main.jsonl")
	if err := os.MkdirAll(filepath.Dir(logfile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logfile, []byte(`{"type":"start"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RemoveEntryLine(dir, `{"type":"gone"}`); err == nil {
		t.Error("expected error when the line isn't present")
	}
}

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
