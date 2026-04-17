package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergePermission(t *testing.T) {
	cases := []struct {
		name        string
		initial     string // empty = file doesn't exist
		entry       string
		wantChanged bool
		wantAllow   []string
		wantExtra   map[string]any // assertions on other top-level keys
	}{
		{
			name:        "file_does_not_exist",
			initial:     "",
			entry:       "Bash(jotter:*)",
			wantChanged: true,
			wantAllow:   []string{"Bash(jotter:*)"},
		},
		{
			name:        "empty_file",
			initial:     "",
			entry:       "Bash(jotter:*)",
			wantChanged: true,
			wantAllow:   []string{"Bash(jotter:*)"},
		},
		{
			name:        "no_permissions_key",
			initial:     `{"theme": "dark"}`,
			entry:       "Bash(jotter:*)",
			wantChanged: true,
			wantAllow:   []string{"Bash(jotter:*)"},
			wantExtra:   map[string]any{"theme": "dark"},
		},
		{
			name:        "permissions_exists_allow_missing",
			initial:     `{"permissions": {"deny": ["Bash(rm:*)"]}}`,
			entry:       "Bash(jotter:*)",
			wantChanged: true,
			wantAllow:   []string{"Bash(jotter:*)"},
		},
		{
			name:        "existing_allow_list",
			initial:     `{"permissions": {"allow": ["Bash(git:*)"]}}`,
			entry:       "Bash(jotter:*)",
			wantChanged: true,
			wantAllow:   []string{"Bash(git:*)", "Bash(jotter:*)"},
		},
		{
			name:        "entry_already_present",
			initial:     `{"permissions": {"allow": ["Bash(jotter:*)", "Bash(git:*)"]}}`,
			entry:       "Bash(jotter:*)",
			wantChanged: false,
			wantAllow:   []string{"Bash(jotter:*)", "Bash(git:*)"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "settings.json")

			if tc.initial != "" {
				if err := os.WriteFile(path, []byte(tc.initial), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			changed, err := MergePermission(path, tc.entry)
			if err != nil {
				t.Fatalf("MergePermission: %v", err)
			}
			if changed != tc.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tc.wantChanged)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading back: %v", err)
			}
			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("parsing written file: %v", err)
			}

			permissions, _ := got["permissions"].(map[string]any)
			allow, _ := permissions["allow"].([]any)
			if len(allow) != len(tc.wantAllow) {
				t.Fatalf("allow has %d entries, want %d — got %v", len(allow), len(tc.wantAllow), allow)
			}
			for i, want := range tc.wantAllow {
				if allow[i] != want {
					t.Errorf("allow[%d] = %v, want %q", i, allow[i], want)
				}
			}

			for k, want := range tc.wantExtra {
				if got[k] != want {
					t.Errorf("key %q = %v, want %v", k, got[k], want)
				}
			}

			// Rewrite should be idempotent.
			again, err := MergePermission(path, tc.entry)
			if err != nil {
				t.Fatalf("second MergePermission: %v", err)
			}
			if again {
				t.Errorf("second merge reported changed=true, want false (idempotent)")
			}
		})
	}
}

func TestMergePermissionMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := MergePermission(path, "Bash(jotter:*)")
	if err == nil {
		t.Fatal("expected error on malformed JSON, got nil")
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("error should mention parsing; got: %v", err)
	}
}
