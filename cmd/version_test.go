package cmd

import (
	"runtime"
	"strings"
	"testing"
)

func TestVersionStringDefaults(t *testing.T) {
	got := versionString()

	for _, want := range []string{"jotter dev", "commit: none", "built:  unknown", runtime.Version()} {
		if !strings.Contains(got, want) {
			t.Errorf("versionString() missing %q; got:\n%s", want, got)
		}
	}
}

func TestVersionStringInjected(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	t.Cleanup(func() {
		version, commit, date = oldVersion, oldCommit, oldDate
	})

	version = "v1.2.3"
	commit = "abc123"
	date = "2026-04-17T12:00:00Z"

	got := versionString()
	for _, want := range []string{"jotter v1.2.3", "commit: abc123", "built:  2026-04-17T12:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("versionString() with injected values missing %q; got:\n%s", want, got)
		}
	}
}
