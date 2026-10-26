package cmd_test

import (
	"encoding/json"
	"strings"
	"testing"
)

type jsonEntry struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Branch    string `json:"branch"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Next      string `json:"next"`
}

func decodeJSON[T any](t *testing.T, stdout string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(stdout), &v); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	return v
}

func TestFormat_RejectsUnknownValue(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	for _, args := range [][]string{
		{"tail", "--project", "proj-a", "--branch", "main"},
		{"search", "--all"},
		{"ls", "--all"},
	} {
		_, stderr, code := runJotter(t, dir, append(args, "--format", "yaml")...)
		if code != 1 {
			t.Errorf("%s: exit code %d, want 1", args[0], code)
		}
		if !strings.Contains(stderr, "md") || !strings.Contains(stderr, "json") {
			t.Errorf("%s: stderr should name the valid formats: %s", args[0], stderr)
		}
	}
}

func TestFormat_MdIsTheDefault(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	def, _, _ := runJotter(t, dir, "tail", "--project", "proj-a", "--branch", "main")
	md, _, _ := runJotter(t, dir, "tail", "--project", "proj-a", "--branch", "main", "--format", "md")
	if def != md {
		t.Errorf("default output differs from --format md:\n%s\n---\n%s", def, md)
	}
}

func TestTail_FormatJSON(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	runJotter(t, dir, "write", "--project", "proj-a", "--branch", "main",
		"--type", "stop", "--content", "Done", "--next", "Ship it")

	stdout, _, code := runJotter(t, dir, "tail", "--project", "proj-a", "--branch", "main", "-n", "2", "--format", "json")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	got := decodeJSON[[]jsonEntry](t, stdout)
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2: %s", len(got), stdout)
	}
	last := got[1]
	if last.Type != "stop" || last.Content != "Done" || last.Next != "Ship it" {
		t.Errorf("last entry = %+v", last)
	}
	if last.Project != "proj-a" || last.Branch != "main" {
		t.Errorf("project/branch = %q/%q", last.Project, last.Branch)
	}
	if last.Timestamp == "" {
		t.Error("timestamp missing")
	}
	if want := idForType(t, dir, "proj-a", "main", "stop"); last.ID != want {
		t.Errorf("id = %q, want %q", last.ID, want)
	}
}

func TestTail_FormatJSONOmitsEmptyNext(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, _ := runJotter(t, dir, "tail", "--project", "proj-a", "--branch", "main", "--format", "json")
	if strings.Contains(stdout, `"next"`) {
		t.Errorf("next should be omitted when empty: %s", stdout)
	}
}

func TestTail_FormatJSONHasNoColourCodes(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, _ := runJotter(t, dir, "tail", "--project", "proj-a", "--branch", "main", "--format", "json")
	if strings.Contains(stdout, "\x1b[") {
		t.Errorf("JSON output contains ANSI escapes: %q", stdout)
	}
}

func TestTail_FormatJSONKeepsAngleBracketsLiteral(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir, "write", "--project", "proj", "--branch", "main",
		"--type", "note", "--content", "line -> commit <id> & more")
	stdout, _, _ := runJotter(t, dir, "tail", "--project", "proj", "--branch", "main", "--format", "json")
	if !strings.Contains(stdout, "line -> commit <id> & more") {
		t.Errorf("content should not be HTML-escaped: %s", stdout)
	}
}

func TestSearch_FormatJSON(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	runJotter(t, dir, "write", "--project", "proj-a", "--branch", "feature/login",
		"--type", "note", "--content", "auth on a slashed branch")

	stdout, _, code := runJotter(t, dir, "search", "--all", "auth", "--format", "json")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	got := decodeJSON[[]jsonEntry](t, stdout)
	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3: %s", len(got), stdout)
	}
	seen := map[string]jsonEntry{}
	for _, e := range got {
		seen[e.Project+"/"+e.Branch] = e
		if e.ID == "" {
			t.Errorf("entry missing id: %+v", e)
		}
	}
	for _, key := range []string{"proj-a/main", "proj-b/main", "proj-a/feature/login"} {
		if _, ok := seen[key]; !ok {
			t.Errorf("missing result for %s: %v", key, got)
		}
	}
	if e := seen["proj-b/main"]; e.Next != "Monitor error rates" {
		t.Errorf("next = %q", e.Next)
	}
}

func TestLs_FormatJSONProjects(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, code := runJotter(t, dir, "ls", "--all", "--format", "json")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	got := decodeJSON[[]struct {
		Name string `json:"name"`
		Last string `json:"last"`
	}](t, stdout)
	if len(got) != 2 {
		t.Fatalf("got %d projects, want 2: %s", len(got), stdout)
	}
	for _, p := range got {
		if p.Name == "" || p.Last == "" {
			t.Errorf("incomplete project: %+v", p)
		}
	}
}

func TestLs_FormatJSONBranches(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, code := runJotter(t, dir, "ls", "--project", "proj-a", "--format", "json")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	got := decodeJSON[[]struct {
		Name    string `json:"name"`
		Entries int    `json:"entries"`
		Last    string `json:"last"`
	}](t, stdout)
	counts := map[string]int{}
	for _, b := range got {
		counts[b.Name] = b.Entries
		if b.Last == "" {
			t.Errorf("branch %s missing last", b.Name)
		}
	}
	if counts["main"] != 2 || counts["feature"] != 1 {
		t.Errorf("branch counts = %v", counts)
	}
}

func TestLs_FormatJSONEntries(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, code := runJotter(t, dir, "ls", "--project", "proj-a", "--branch", "main", "--format", "json")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	got := decodeJSON[[]struct {
		Timestamp string `json:"timestamp"`
		Type      string `json:"type"`
		Title     string `json:"title"`
	}](t, stdout)
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2: %s", len(got), stdout)
	}
	if got[0].Type != "checkpoint" || got[0].Title != "Added token refresh logic" {
		t.Errorf("newest entry first, got %+v", got[0])
	}
}
