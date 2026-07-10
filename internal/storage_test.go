package internal

import (
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
