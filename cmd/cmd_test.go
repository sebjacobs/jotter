package cmd_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// buildBinary builds the jotter binary and returns the path.
// It's built once per test run via TestMain or called per-test.
var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary into a temp location
	tmp, err := os.MkdirTemp("", "jotter-test-bin")
	if err != nil {
		panic(err)
	}
	binaryPath = filepath.Join(tmp, "jotter")
	cmd := exec.Command("go", "build", "-o", binaryPath, "github.com/sebjacobs/jotter")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("failed to build binary: " + string(out))
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// initDataDir creates a git-initialized temp directory for tests.
func initDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	logsDir := filepath.Join(dir, "logs")
	os.MkdirAll(logsDir, 0o755)
	os.WriteFile(filepath.Join(logsDir, ".gitkeep"), []byte{}, 0o644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.CombinedOutput()
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = dir
	cmd.CombinedOutput()
	return dir
}

// runJotter runs the jotter binary with args and JOTTER_DATA set.
func runJotter(t *testing.T, dataDir string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "JOTTER_DATA="+dataDir)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run jotter: %v", err)
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

// ---------------------------------------------------------------------------
// write command
// ---------------------------------------------------------------------------

func TestWrite_CreatesFileAndDirectories(t *testing.T) {
	dir := initDataDir(t)
	stdout, _, code := runJotter(t, dir,
		"write", "--project", "new-proj", "--branch", "feature-x",
		"--type", "start", "--content", "First entry")
	if code != 0 {
		t.Fatalf("exit code %d, stdout: %s", code, stdout)
	}

	path := filepath.Join(dir, "logs", "new-proj", "feature-x.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	var entry map[string]interface{}
	json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry)
	if entry["type"] != "start" {
		t.Errorf("type = %v", entry["type"])
	}
	if entry["content"] != "First entry" {
		t.Errorf("content = %v", entry["content"])
	}
}

func TestWrite_ConfirmationMessage(t *testing.T) {
	dir := initDataDir(t)
	stdout, _, _ := runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "checkpoint", "--content", "Some work")
	if !strings.Contains(stdout, "Wrote checkpoint entry to logs/proj/main.jsonl") {
		t.Errorf("unexpected stdout: %s", stdout)
	}
}

func TestWrite_ISOTimestamp(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "start", "--content", "Check timestamp")

	data, _ := os.ReadFile(filepath.Join(dir, "logs", "proj", "main.jsonl"))
	var entry map[string]interface{}
	json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry)
	ts, ok := entry["timestamp"].(string)
	if !ok {
		t.Fatal("no timestamp")
	}
	if !regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`).MatchString(ts) {
		t.Errorf("timestamp %q doesn't match ISO format", ts)
	}
}

func TestWrite_NextFieldPresent(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "finish", "--content", "Done",
		"--next", "Pick up testing")

	data, _ := os.ReadFile(filepath.Join(dir, "logs", "proj", "main.jsonl"))
	var entry map[string]interface{}
	json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry)
	if entry["next"] != "Pick up testing" {
		t.Errorf("next = %v", entry["next"])
	}
}

func TestWrite_NextFieldAbsent(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "start", "--content", "Starting up")

	data, _ := os.ReadFile(filepath.Join(dir, "logs", "proj", "main.jsonl"))
	var entry map[string]interface{}
	json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry)
	if _, ok := entry["next"]; ok {
		t.Error("next field should be absent")
	}
}

func TestWrite_AppendsToExistingFile(t *testing.T) {
	dir := initDataDir(t)
	for i := range 3 {
		runJotter(t, dir,
			"write", "--project", "proj", "--branch", "main",
			"--type", "checkpoint", "--content", "Entry "+string(rune('0'+i)))
	}

	data, _ := os.ReadFile(filepath.Join(dir, "logs", "proj", "main.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
}

func TestWrite_InvalidTypeRejected(t *testing.T) {
	dir := initDataDir(t)
	_, stderr, code := runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "invalid", "--content", "Nope")
	if code == 0 {
		t.Error("expected non-zero exit code for invalid type")
	}
	if !strings.Contains(stderr, "invalid") {
		t.Errorf("stderr should mention invalid type: %s", stderr)
	}
}

func TestWrite_MultilineContent(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "checkpoint", "--content", "Line one\nLine two\nLine three")

	data, _ := os.ReadFile(filepath.Join(dir, "logs", "proj", "main.jsonl"))
	var entry map[string]interface{}
	json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry)
	if entry["content"] != "Line one\nLine two\nLine three" {
		t.Errorf("content = %v", entry["content"])
	}
}

func TestWrite_CommitsToDataRepo(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "start", "--content", "Should be committed")

	cmd := exec.Command("git", "log", "--oneline")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "session: proj/main start") {
		t.Errorf("git log doesn't contain expected commit: %s", out)
	}
}

func TestWrite_CommitMessageFormat(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir,
		"write", "--project", "my-app", "--branch", "feature-x",
		"--type", "checkpoint", "--content", "Progress")

	cmd := exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if !strings.HasPrefix(msg, "session: my-app/feature-x checkpoint") {
		t.Errorf("commit message = %q", msg)
	}
}

// ---------------------------------------------------------------------------
// tail command
// ---------------------------------------------------------------------------

func TestTail_ReturnsLastEntry(t *testing.T) {
	dir := initDataDir(t)
	for _, typ := range []string{"start", "checkpoint", "finish"} {
		runJotter(t, dir,
			"write", "--project", "proj", "--branch", "main",
			"--type", typ, "--content", "Content for "+typ)
	}
	stdout, _, code := runJotter(t, dir,
		"tail", "--project", "proj", "--branch", "main")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "| finish") {
		t.Errorf("should contain finish heading: %s", stdout)
	}
	if !strings.Contains(stdout, "Content for finish") {
		t.Errorf("should contain finish content: %s", stdout)
	}
	if strings.Contains(stdout, "Content for start") {
		t.Error("should not contain start content")
	}
}

func TestTail_MissingFileExitsWithError(t *testing.T) {
	dir := initDataDir(t)
	_, stderr, code := runJotter(t, dir,
		"tail", "--project", "nope", "--branch", "nope")
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
	if !strings.Contains(stderr, "No log file") {
		t.Errorf("stderr: %s", stderr)
	}
}

func TestTail_RendersNextField(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "finish", "--content", "Wrapped up",
		"--next", "Continue tomorrow")
	stdout, _, _ := runJotter(t, dir,
		"tail", "--project", "proj", "--branch", "main")
	if !strings.Contains(stdout, "**Next:** Continue tomorrow") {
		t.Errorf("missing next field: %s", stdout)
	}
}

func TestTail_LimitReturnsMultiple(t *testing.T) {
	dir := initDataDir(t)
	for i := range 5 {
		runJotter(t, dir,
			"write", "--project", "proj", "--branch", "main",
			"--type", "checkpoint", "--content", fmt.Sprintf("Entry %d", i))
	}
	stdout, _, code := runJotter(t, dir,
		"tail", "--project", "proj", "--branch", "main",
		"--limit", "3")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "Entry 2") || !strings.Contains(stdout, "Entry 3") || !strings.Contains(stdout, "Entry 4") {
		t.Errorf("missing expected entries: %s", stdout)
	}
	if strings.Contains(stdout, "Entry 1") || strings.Contains(stdout, "Entry 0") {
		t.Errorf("should not contain older entries: %s", stdout)
	}
}

func TestTail_LimitExceedingCountReturnsAll(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "start", "--content", "Only entry")
	stdout, _, code := runJotter(t, dir,
		"tail", "--project", "proj", "--branch", "main",
		"--limit", "10")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "Only entry") {
		t.Errorf("missing entry: %s", stdout)
	}
}

func TestTail_TimestampInHeading(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "start", "--content", "Hello")
	stdout, _, _ := runJotter(t, dir,
		"tail", "--project", "proj", "--branch", "main")
	if !regexp.MustCompile(`## \d{4}-\d{2}-\d{2} \d{2}:\d{2} \| start`).MatchString(stdout) {
		t.Errorf("timestamp heading not found: %s", stdout)
	}
}

// ---------------------------------------------------------------------------
// ls command
// ---------------------------------------------------------------------------

func TestLs_ListsProjects(t *testing.T) {
	dir := initDataDir(t)
	for _, project := range []string{"alpha", "beta", "gamma"} {
		runJotter(t, dir,
			"write", "--project", project, "--branch", "main",
			"--type", "start", "--content", "hello")
	}
	stdout, _, code := runJotter(t, dir, "ls")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %s", len(lines), stdout)
	}
	names := make([]string, len(lines))
	for i, line := range lines {
		names[i] = strings.Fields(line)[0]
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, []string{"alpha", "beta", "gamma"}) {
		t.Errorf("project names = %v", names)
	}
	for _, line := range lines {
		if !strings.Contains(line, "(last:") {
			t.Errorf("missing last activity: %s", line)
		}
	}
}

func TestLs_ListsBranchesForProject(t *testing.T) {
	dir := initDataDir(t)
	for _, branch := range []string{"main", "feature-auth", "feature-ui"} {
		runJotter(t, dir,
			"write", "--project", "my-app", "--branch", branch,
			"--type", "start", "--content", "hello")
	}
	stdout, _, code := runJotter(t, dir, "ls", "--project", "my-app")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "feature-auth") || !strings.Contains(stdout, "feature-ui") || !strings.Contains(stdout, "main") {
		t.Errorf("missing branches: %s", stdout)
	}
}

func TestBranchSanitisation_SlashConvertedToPlus(t *testing.T) {
	dir := initDataDir(t)
	stdout, _, code := runJotter(t, dir,
		"write", "--project", "proj", "--branch", "feature/auth",
		"--type", "start", "--content", "hello")
	if code != 0 {
		t.Fatalf("exit code %d, stdout: %s", code, stdout)
	}
	// File on disk uses + separator
	path := filepath.Join(dir, "logs", "proj", "feature+auth.jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}
	// tail reads back via the slashed branch name
	tailOut, _, tailCode := runJotter(t, dir,
		"tail", "--project", "proj", "--branch", "feature/auth")
	if tailCode != 0 {
		t.Fatalf("tail exit code %d", tailCode)
	}
	if !strings.Contains(tailOut, "hello") {
		t.Errorf("tail should return the entry, got: %s", tailOut)
	}
	// ls displays the original / form
	lsOut, _, lsCode := runJotter(t, dir, "ls", "--project", "proj")
	if lsCode != 0 {
		t.Fatalf("ls exit code %d", lsCode)
	}
	if !strings.Contains(lsOut, "feature/auth") {
		t.Errorf("ls should show original branch name, got: %s", lsOut)
	}
	if strings.Contains(lsOut, "feature+auth") {
		t.Errorf("ls should not show sanitised name, got: %s", lsOut)
	}
}

func TestLs_BranchListingShowsCountAndDate(t *testing.T) {
	dir := initDataDir(t)
	for range 3 {
		runJotter(t, dir,
			"write", "--project", "proj", "--branch", "main",
			"--type", "checkpoint", "--content", "entry")
	}
	stdout, _, _ := runJotter(t, dir, "ls", "--project", "proj")
	if !strings.Contains(stdout, "3 entries") {
		t.Errorf("missing entry count: %s", stdout)
	}
	if !strings.Contains(stdout, "last:") {
		t.Errorf("missing last date: %s", stdout)
	}
}

func TestLs_NoProjectsExitsWithError(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "logs"), 0o755)
	_, _, code := runJotter(t, dir, "ls")
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestLs_UnknownProjectExitsWithError(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "logs"), 0o755)
	_, _, code := runJotter(t, dir, "ls", "--project", "nonexistent")
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

// ---------------------------------------------------------------------------
// search command
// ---------------------------------------------------------------------------

// populateSearchData creates entries across multiple projects/branches for search tests.
func populateSearchData(t *testing.T, dir string) {
	t.Helper()
	runs := []struct {
		project, branch, typ, content, next string
	}{
		{"proj-a", "main", "start", "Implementing auth flow", ""},
		{"proj-a", "main", "checkpoint", "Added token refresh logic", ""},
		{"proj-a", "feature", "start", "Setting up database migrations", ""},
		{"proj-b", "main", "finish", "Deployed auth service", "Monitor error rates"},
	}
	for _, r := range runs {
		args := []string{
			"write", "--project", r.project, "--branch", r.branch,
			"--type", r.typ, "--content", r.content,
		}
		if r.next != "" {
			args = append(args, "--next", r.next)
		}
		runJotter(t, dir, args...)
	}
}

func TestSearch_ByTerm(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, code := runJotter(t, dir, "search", "auth")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "auth flow") || !strings.Contains(stdout, "auth service") {
		t.Errorf("missing auth entries: %s", stdout)
	}
	if strings.Contains(stdout, "database migrations") {
		t.Errorf("should not contain non-matching entry: %s", stdout)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, code := runJotter(t, dir, "search", "AUTH")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "auth flow") {
		t.Errorf("case-insensitive search failed: %s", stdout)
	}
}

func TestSearch_IncludesNextField(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, code := runJotter(t, dir, "search", "error rates")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "Deployed auth service") {
		t.Errorf("should find entry via next field: %s", stdout)
	}
}

func TestSearch_ScopedByProject(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, code := runJotter(t, dir, "search", "auth", "--project", "proj-a")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "auth flow") {
		t.Errorf("missing proj-a entry: %s", stdout)
	}
	if strings.Contains(stdout, "auth service") {
		t.Errorf("should not contain proj-b entry: %s", stdout)
	}
}

func TestSearch_ScopedByProjectAndBranch(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, code := runJotter(t, dir, "search", "auth", "--project", "proj-a", "--branch", "main")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "auth flow") {
		t.Errorf("missing entry: %s", stdout)
	}
	if strings.Contains(stdout, "database migrations") {
		t.Errorf("should not contain other branch entry: %s", stdout)
	}
}

func TestSearch_ScopedByType(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, code := runJotter(t, dir, "search", "auth", "--type", "finish")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "Deployed auth service") {
		t.Errorf("missing finish entry: %s", stdout)
	}
	if strings.Contains(stdout, "auth flow") {
		t.Errorf("should not contain non-finish entries: %s", stdout)
	}
}

func TestSearch_ScopedBySince(t *testing.T) {
	dir := initDataDir(t)
	// Write entries with known timestamps directly
	os.MkdirAll(filepath.Join(dir, "logs", "proj"), 0o755)
	jsonlFile := filepath.Join(dir, "logs", "proj", "main.jsonl")
	old := `{"timestamp":"2026-01-01T10:00:00","type":"start","content":"Old auth work"}`
	recent := `{"timestamp":"2026-04-10T10:00:00","type":"checkpoint","content":"Recent auth work"}`
	os.WriteFile(jsonlFile, []byte(old+"\n"+recent+"\n"), 0o644)

	stdout, _, code := runJotter(t, dir, "search", "auth", "--since", "2026-04-01")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "Recent auth work") {
		t.Errorf("missing recent entry: %s", stdout)
	}
	if strings.Contains(stdout, "Old auth work") {
		t.Errorf("should not contain old entry: %s", stdout)
	}
}

func TestSearch_WithoutTermReturnsAll(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, code := runJotter(t, dir, "search", "--type", "finish")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "Deployed auth service") {
		t.Errorf("missing entry: %s", stdout)
	}
}

func TestSearch_ProvenancePrefix(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	stdout, _, code := runJotter(t, dir, "search", "database")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "[proj-a/feature.jsonl]") {
		t.Errorf("missing provenance prefix: %s", stdout)
	}
}

func TestSearch_NoResults(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	_, _, code := runJotter(t, dir, "search", "nonexistent term")
	if code == 0 {
		t.Error("expected non-zero exit code for no results")
	}
}
