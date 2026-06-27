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

// runJotter runs the jotter binary from a fresh working directory with a
// .jotter file pointing at dataDir. HOME is redirected to a clean sandbox so
// the real user's ~/.jotter (if any) cannot leak in.
func runJotter(t *testing.T, dataDir string, args ...string) (string, string, int) {
	t.Helper()
	workdir := t.TempDir()
	configPath := filepath.Join(workdir, ".jotter")
	body := fmt.Sprintf("data_dir = %q\n", dataDir)
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cleanHome := t.TempDir()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "HOME="+cleanHome)
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
	var entry map[string]any
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
	var entry map[string]any
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
	var entry map[string]any
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
	var entry map[string]any
	json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry)
	if _, ok := entry["next"]; ok {
		t.Error("next field should be absent")
	}
}

func TestWrite_FinishCommitsLocallyWithoutPushing(t *testing.T) {
	dir, bare := initDataDirWithRemote(t)
	remoteBefore := git(t, bare, "log", "--oneline")

	stdout, stderr, code := runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "finish", "--content", "Done")
	if code != 0 {
		t.Fatalf("exit code %d, stdout: %s stderr: %s", code, stdout, stderr)
	}
	if strings.Contains(stderr, "push") {
		t.Errorf("finish should not push inline, got stderr: %s", stderr)
	}

	if local := git(t, dir, "log", "--oneline"); !strings.Contains(local, "session: proj/main finish") {
		t.Errorf("finish entry was not committed locally: %s", local)
	}
	if remoteAfter := git(t, bare, "log", "--oneline"); remoteAfter != remoteBefore {
		t.Errorf("finish pushed inline — remote changed:\nbefore: %s\nafter:  %s", remoteBefore, remoteAfter)
	}
}

func TestWrite_StopCommitsLocallyWithoutPushing(t *testing.T) {
	dir, bare := initDataDirWithRemote(t)
	remoteBefore := git(t, bare, "log", "--oneline")

	stdout, stderr, code := runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "stop", "--content", "Done", "--next", "Pick up testing")
	if code != 0 {
		t.Fatalf("exit code %d, stdout: %s stderr: %s", code, stdout, stderr)
	}
	if strings.Contains(stderr, "push") {
		t.Errorf("stop should not push inline, got stderr: %s", stderr)
	}

	if local := git(t, dir, "log", "--oneline"); !strings.Contains(local, "session: proj/main stop") {
		t.Errorf("stop entry was not committed locally: %s", local)
	}
	if remoteAfter := git(t, bare, "log", "--oneline"); remoteAfter != remoteBefore {
		t.Errorf("stop pushed inline — remote changed:\nbefore: %s\nafter:  %s", remoteBefore, remoteAfter)
	}
}

func TestWrite_HandoverCommitsLocallyWithoutPushing(t *testing.T) {
	dir, bare := initDataDirWithRemote(t)
	remoteBefore := git(t, bare, "log", "--oneline")

	stdout, stderr, code := runJotter(t, dir,
		"write", "--project", "proj", "--branch", "main",
		"--type", "handover", "--content", "From feature/auth — OAuth shipped",
		"--next", "Add refresh-token support")
	if code != 0 {
		t.Fatalf("exit code %d, stdout: %s stderr: %s", code, stdout, stderr)
	}
	if strings.Contains(stderr, "push") {
		t.Errorf("handover should not push inline, got stderr: %s", stderr)
	}

	if local := git(t, dir, "log", "--oneline"); !strings.Contains(local, "session: proj/main handover") {
		t.Errorf("handover entry was not committed locally: %s", local)
	}
	if remoteAfter := git(t, bare, "log", "--oneline"); remoteAfter != remoteBefore {
		t.Errorf("handover pushed inline — remote changed:\nbefore: %s\nafter:  %s", remoteBefore, remoteAfter)
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
	var entry map[string]any
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
	if !strings.Contains(stdout, "Next: Continue tomorrow") {
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

func TestTail_NShorthandAliasesLimit(t *testing.T) {
	dir := initDataDir(t)
	for i := range 5 {
		runJotter(t, dir,
			"write", "--project", "proj", "--branch", "main",
			"--type", "checkpoint", "--content", fmt.Sprintf("Entry %d", i))
	}
	stdout, _, code := runJotter(t, dir,
		"tail", "--project", "proj", "--branch", "main",
		"-n", "3")
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

func TestLs_EntriesForBranch(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir, "write", "--project", "proj", "--branch", "feature/x",
		"--type", "start", "--content", "**Kickoff** for feature x\n\nmore detail")
	runJotter(t, dir, "write", "--project", "proj", "--branch", "feature/x",
		"--type", "checkpoint", "--content", "## Progress update\nwip")

	stdout, _, code := runJotter(t, dir,
		"ls", "--project", "proj", "--branch", "feature/x")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, stdout)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %s", len(lines), stdout)
	}
	if !strings.Contains(lines[0], "checkpoint") || !strings.Contains(lines[0], "Progress update") {
		t.Errorf("first line missing title/type: %s", lines[0])
	}
	if !strings.Contains(lines[1], "start") || !strings.Contains(lines[1], "Kickoff for feature x") {
		t.Errorf("second line missing title/type: %s", lines[1])
	}
	if !regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}`).MatchString(lines[0]) {
		t.Errorf("missing timestamp: %s", lines[0])
	}
}

func TestLs_BranchWithoutProjectExitsWithError(t *testing.T) {
	dir := initDataDir(t)
	_, _, code := runJotter(t, dir, "ls", "--branch", "main")
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestLs_UnknownBranchExitsWithError(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir, "write", "--project", "proj", "--branch", "main",
		"--type", "start", "--content", "hi")
	_, _, code := runJotter(t, dir, "ls", "--project", "proj", "--branch", "nope")
	if code == 0 {
		t.Error("expected non-zero exit code")
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

// writeRawJSONL drops pre-formed JSONL lines into logs/<project>/<branch>.jsonl
// so tests can pin entry timestamps to exact values.
func writeRawJSONL(t *testing.T, dir, project, branch string, lines ...string) {
	t.Helper()
	projectDir := filepath.Join(dir, "logs", project)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(projectDir, branch+".jsonl")
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLs_ProjectsFilteredByWindow(t *testing.T) {
	dir := initDataDir(t)
	writeRawJSONL(t, dir, "proj-a", "main",
		`{"timestamp":"2026-04-10T09:00:00","type":"start","content":"A work"}`,
		`{"timestamp":"2026-04-19T15:00:00","type":"checkpoint","content":"A recent"}`,
	)
	writeRawJSONL(t, dir, "proj-b", "main",
		`{"timestamp":"2026-01-05T10:00:00","type":"start","content":"B old only"}`,
	)
	writeRawJSONL(t, dir, "proj-c", "main",
		`{"timestamp":"2026-04-19T20:00:00","type":"finish","content":"C recent only"}`,
	)

	stdout, _, code := runJotter(t, dir, "ls",
		"--since", "2026-04-19", "--until", "2026-04-19")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, stdout)
	}
	if !strings.Contains(stdout, "proj-a") {
		t.Errorf("proj-a should be listed (has in-window entry): %s", stdout)
	}
	if !strings.Contains(stdout, "proj-c") {
		t.Errorf("proj-c should be listed: %s", stdout)
	}
	if strings.Contains(stdout, "proj-b") {
		t.Errorf("proj-b should be filtered out (no in-window entries): %s", stdout)
	}
	// last: should reflect the in-window timestamp, not the overall last.
	if !strings.Contains(stdout, "2026-04-19 15:00") {
		t.Errorf("proj-a last: should be in-window (15:00), got: %s", stdout)
	}
}

func TestLs_BranchesFilteredByWindow(t *testing.T) {
	dir := initDataDir(t)
	writeRawJSONL(t, dir, "proj", "main",
		`{"timestamp":"2026-04-19T10:00:00","type":"start","content":"main in window"}`,
	)
	writeRawJSONL(t, dir, "proj", "old-feature",
		`{"timestamp":"2026-01-01T10:00:00","type":"start","content":"too old"}`,
	)

	stdout, _, code := runJotter(t, dir, "ls",
		"--project", "proj", "--since", "2026-04-19")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, stdout)
	}
	if !strings.Contains(stdout, "main") {
		t.Errorf("main branch should be listed: %s", stdout)
	}
	if strings.Contains(stdout, "old-feature") {
		t.Errorf("old-feature should be filtered out: %s", stdout)
	}
}

func TestLs_EntriesFilteredByWindow(t *testing.T) {
	dir := initDataDir(t)
	writeRawJSONL(t, dir, "proj", "main",
		`{"timestamp":"2026-04-10T09:00:00","type":"start","content":"Old morning"}`,
		`{"timestamp":"2026-04-19T12:00:00","type":"checkpoint","content":"Target noon"}`,
		`{"timestamp":"2026-04-19T18:00:00","type":"finish","content":"Target evening"}`,
	)

	stdout, _, code := runJotter(t, dir, "ls",
		"--project", "proj", "--branch", "main",
		"--since", "2026-04-19T13:00:00",
		"--until", "2026-04-19T23:59:59")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, stdout)
	}
	if !strings.Contains(stdout, "Target evening") {
		t.Errorf("missing target evening entry: %s", stdout)
	}
	if strings.Contains(stdout, "Old morning") || strings.Contains(stdout, "Target noon") {
		t.Errorf("should only contain the evening entry: %s", stdout)
	}
}

func TestLs_EmptyWindowExitsWithError(t *testing.T) {
	dir := initDataDir(t)
	writeRawJSONL(t, dir, "proj", "main",
		`{"timestamp":"2026-01-01T10:00:00","type":"start","content":"ancient"}`,
	)

	_, stderr, code := runJotter(t, dir, "ls", "--since", "2026-04-19")
	if code == 0 {
		t.Error("expected non-zero exit code for empty window")
	}
	if !strings.Contains(stderr, "in window") {
		t.Errorf("expected error mentioning window: %s", stderr)
	}
}

func TestLs_InvalidBoundary(t *testing.T) {
	dir := initDataDir(t)
	_, stderr, code := runJotter(t, dir, "ls", "--since", "yesterday")
	if code == 0 {
		t.Error("expected non-zero exit code for invalid --since")
	}
	if !strings.Contains(stderr, "invalid --since") {
		t.Errorf("missing error message: %s", stderr)
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

func TestSearch_ScopedByUntil(t *testing.T) {
	dir := initDataDir(t)
	os.MkdirAll(filepath.Join(dir, "logs", "proj"), 0o755)
	jsonlFile := filepath.Join(dir, "logs", "proj", "main.jsonl")
	old := `{"timestamp":"2026-01-01T10:00:00","type":"start","content":"Old auth work"}`
	recent := `{"timestamp":"2026-04-10T10:00:00","type":"checkpoint","content":"Recent auth work"}`
	os.WriteFile(jsonlFile, []byte(old+"\n"+recent+"\n"), 0o644)

	stdout, _, code := runJotter(t, dir, "search", "auth", "--until", "2026-03-01")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "Old auth work") {
		t.Errorf("missing old entry: %s", stdout)
	}
	if strings.Contains(stdout, "Recent auth work") {
		t.Errorf("should not contain recent entry: %s", stdout)
	}
}

func TestSearch_ScopedBySingleDay(t *testing.T) {
	dir := initDataDir(t)
	os.MkdirAll(filepath.Join(dir, "logs", "proj"), 0o755)
	jsonlFile := filepath.Join(dir, "logs", "proj", "main.jsonl")
	before := `{"timestamp":"2026-04-09T23:59:59","type":"checkpoint","content":"Day before entry"}`
	morning := `{"timestamp":"2026-04-10T08:00:00","type":"start","content":"Target day morning"}`
	evening := `{"timestamp":"2026-04-10T20:00:00","type":"finish","content":"Target day evening"}`
	after := `{"timestamp":"2026-04-11T00:00:01","type":"note","content":"Day after entry"}`
	os.WriteFile(jsonlFile, []byte(before+"\n"+morning+"\n"+evening+"\n"+after+"\n"), 0o644)

	stdout, _, code := runJotter(t, dir, "search", "--since", "2026-04-10", "--until", "2026-04-10")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "Target day morning") || !strings.Contains(stdout, "Target day evening") {
		t.Errorf("missing target day entries: %s", stdout)
	}
	if strings.Contains(stdout, "Day before entry") || strings.Contains(stdout, "Day after entry") {
		t.Errorf("should only contain target day: %s", stdout)
	}
}

func TestSearch_ScopedByTimestamp(t *testing.T) {
	dir := initDataDir(t)
	os.MkdirAll(filepath.Join(dir, "logs", "proj"), 0o755)
	jsonlFile := filepath.Join(dir, "logs", "proj", "main.jsonl")
	morning := `{"timestamp":"2026-04-10T09:00:00","type":"start","content":"Morning entry"}`
	noon := `{"timestamp":"2026-04-10T12:00:00","type":"checkpoint","content":"Noon entry"}`
	evening := `{"timestamp":"2026-04-10T18:00:00","type":"finish","content":"Evening entry"}`
	os.WriteFile(jsonlFile, []byte(morning+"\n"+noon+"\n"+evening+"\n"), 0o644)

	stdout, _, code := runJotter(t, dir, "search",
		"--since", "2026-04-10T10:00:00",
		"--until", "2026-04-10T15:00:00")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "Noon entry") {
		t.Errorf("missing noon entry: %s", stdout)
	}
	if strings.Contains(stdout, "Morning entry") || strings.Contains(stdout, "Evening entry") {
		t.Errorf("should only contain noon entry: %s", stdout)
	}
}

func TestSearch_InvalidBoundary(t *testing.T) {
	dir := initDataDir(t)
	populateSearchData(t, dir)
	_, stderr, code := runJotter(t, dir, "search", "--since", "yesterday")
	if code == 0 {
		t.Error("expected non-zero exit code for invalid --since")
	}
	if !strings.Contains(stderr, "invalid --since") {
		t.Errorf("missing error message: %s", stderr)
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

// ---------------------------------------------------------------------------
// sync command
// ---------------------------------------------------------------------------

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s", args, out)
	}
	return string(out)
}

// initDataDirWithRemote creates a bare remote and a data dir cloned from it, so
// the data dir has an "origin" remote and a tracking branch. Returns
// (dataDir, bareRemote).
func initDataDirWithRemote(t *testing.T) (string, string) {
	t.Helper()
	bare := t.TempDir()
	git(t, bare, "init", "--bare", "-b", "main")

	seed := t.TempDir()
	git(t, seed, "init", "-b", "main")
	git(t, seed, "config", "user.email", "test@test.com")
	git(t, seed, "config", "user.name", "Test")
	os.MkdirAll(filepath.Join(seed, "logs"), 0o755)
	os.WriteFile(filepath.Join(seed, "logs", ".gitkeep"), []byte{}, 0o644)
	git(t, seed, "add", ".")
	git(t, seed, "commit", "-m", "init")
	git(t, seed, "remote", "add", "origin", bare)
	git(t, seed, "push", "-u", "origin", "main")

	dataDir := t.TempDir()
	git(t, dataDir, "clone", bare, dataDir)
	git(t, dataDir, "config", "user.email", "test@test.com")
	git(t, dataDir, "config", "user.name", "Test")
	return dataDir, bare
}

func TestSync_NoRemoteExitsWithError(t *testing.T) {
	dir := initDataDir(t)
	_, stderr, code := runJotter(t, dir, "sync")
	if code == 0 {
		t.Error("expected non-zero exit code with no remote")
	}
	if !strings.Contains(stderr, "no git remote") {
		t.Errorf("stderr missing expected message: %s", stderr)
	}
}

func TestSync_PushesPendingCommits(t *testing.T) {
	dir, bare := initDataDirWithRemote(t)

	runJotter(t, dir, "write", "--project", "proj", "--branch", "main",
		"--type", "checkpoint", "--content", "Unpushed work")

	stdout, stderr, code := runJotter(t, dir, "sync")
	if code != 0 {
		t.Fatalf("exit code %d, stdout: %s stderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Pushed") {
		t.Errorf("stdout should report a push: %s", stdout)
	}

	log := git(t, bare, "log", "--oneline")
	if !strings.Contains(log, "session: proj/main checkpoint") {
		t.Errorf("commit did not reach the remote: %s", log)
	}
}

func TestSync_AlreadyInSync(t *testing.T) {
	dir, _ := initDataDirWithRemote(t)
	runJotter(t, dir, "write", "--project", "proj", "--branch", "main",
		"--type", "checkpoint", "--content", "Work")
	runJotter(t, dir, "sync")

	stdout, _, code := runJotter(t, dir, "sync")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(stdout, "Already in sync") {
		t.Errorf("expected already-in-sync message: %s", stdout)
	}
}

func TestSync_RebasesWhenRemoteMovedOn(t *testing.T) {
	dir, bare := initDataDirWithRemote(t)

	// A second clone pushes a commit, so the remote is ahead of dir.
	other := t.TempDir()
	git(t, other, "clone", bare, other)
	git(t, other, "config", "user.email", "test@test.com")
	git(t, other, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(other, "logs", "remote-change.txt"), []byte("hi"), 0o644)
	git(t, other, "add", ".")
	git(t, other, "commit", "-m", "remote change")
	git(t, other, "push")

	// Meanwhile dir has its own unpushed entry — histories diverge.
	runJotter(t, dir, "write", "--project", "proj", "--branch", "main",
		"--type", "checkpoint", "--content", "Local work")

	stdout, stderr, code := runJotter(t, dir, "sync")
	if code != 0 {
		t.Fatalf("exit code %d, stdout: %s stderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "rebasing") {
		t.Errorf("expected rebase message: %s", stdout)
	}

	log := git(t, bare, "log", "--oneline")
	if !strings.Contains(log, "session: proj/main checkpoint") {
		t.Errorf("local commit did not reach the remote: %s", log)
	}
	if !strings.Contains(log, "remote change") {
		t.Errorf("remote commit missing after rebase: %s", log)
	}
}

// ---------------------------------------------------------------------------
// project / branch commands
// ---------------------------------------------------------------------------

// runJotterFromGitRepo runs jotter from a fresh git-initialised workdir. Lets
// us exercise project/branch detection without the real user's git state.
// Returns (stdout, stderr, exitCode, workdir).
func runJotterFromGitRepo(t *testing.T, branch string, args ...string) (string, string, int, string) {
	t.Helper()
	workdir := t.TempDir()
	for _, cmdArgs := range [][]string{
		{"git", "init", "-b", branch},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	} {
		c := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		c.Dir = workdir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", cmdArgs, out)
		}
	}
	cleanHome := t.TempDir()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "HOME="+cleanHome)
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
	return stdout.String(), stderr.String(), exitCode, workdir
}

// runJotterFromGitRepoWithData runs jotter from a fresh git repo on the given
// branch, with a .jotter file in that repo pointing at dataDir. It returns the
// repo's path so callers can derive the inferred project name (its basename).
func runJotterFromGitRepoWithData(t *testing.T, dataDir, branch string, args ...string) (string, string, int, string) {
	t.Helper()
	workdir := t.TempDir()
	for _, cmdArgs := range [][]string{
		{"git", "init", "-b", branch},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	} {
		c := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		c.Dir = workdir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", cmdArgs, out)
		}
	}
	configPath := filepath.Join(workdir, ".jotter")
	if err := os.WriteFile(configPath, fmt.Appendf(nil, "data_dir = %q\n", dataDir), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, exitCode := runJotterIn(t, workdir, args...)
	return stdout, stderr, exitCode, workdir
}

// runJotterIn runs the jotter binary in an existing working directory with a
// clean HOME, leaving any .jotter / git setup already there in place.
func runJotterIn(t *testing.T, workdir string, args ...string) (string, string, int) {
	t.Helper()
	cleanHome := t.TempDir()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "HOME="+cleanHome)
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

func TestWrite_InfersProjectAndBranchFromGit(t *testing.T) {
	dir := initDataDir(t)
	_, stderr, code, workdir := runJotterFromGitRepoWithData(t, dir, "feature/infer",
		"write", "--type", "note", "--content", "inferred entry")
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr)
	}
	project := filepath.Base(workdir)
	path := filepath.Join(dir, "logs", project, "feature+infer.jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected log file at %s: %v", path, err)
	}
}

func TestTail_InfersProjectAndBranchFromGit(t *testing.T) {
	dir := initDataDir(t)
	_, stderr, code, workdir := runJotterFromGitRepoWithData(t, dir, "feature/infer",
		"write", "--type", "note", "--content", "inferred entry")
	if code != 0 {
		t.Fatalf("write exit code %d, stderr: %s", code, stderr)
	}
	// Tail from the same repo with no --project/--branch should find that entry.
	stdout, stderr, code := runJotterIn(t, workdir, "tail")
	if code != 0 {
		t.Fatalf("tail exit code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "inferred entry") {
		t.Errorf("tail output missing inferred entry: %s", stdout)
	}
}

func TestWrite_ExplicitFlagsOverrideInference(t *testing.T) {
	dir := initDataDir(t)
	_, stderr, code, _ := runJotterFromGitRepoWithData(t, dir, "feature/infer",
		"write", "--project", "explicit-proj", "--branch", "explicit-branch",
		"--type", "note", "--content", "x")
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr)
	}
	path := filepath.Join(dir, "logs", "explicit-proj", "explicit-branch.jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("explicit flags should win, expected file at %s: %v", path, err)
	}
}

func TestWrite_ErrorsOutsideGitRepoWithoutFlags(t *testing.T) {
	dir := initDataDir(t)
	_, stderr, code := runJotter(t, dir, "write", "--type", "note", "--content", "x")
	if code == 0 {
		t.Error("expected non-zero exit outside a git repo without --project/--branch")
	}
	if !strings.Contains(stderr, "not inside a git repo") {
		t.Errorf("stderr missing expected message: %s", stderr)
	}
}

func TestProject_PrintsBasenameOfGitToplevel(t *testing.T) {
	stdout, _, code, workdir := runJotterFromGitRepo(t, "main", "project")
	if code != 0 {
		t.Fatalf("exit code %d, stdout: %s", code, stdout)
	}
	want := filepath.Base(workdir)
	if got := strings.TrimSpace(stdout); got != want {
		t.Errorf("project = %q, want %q", got, want)
	}
}

func TestProject_ReturnsMainRepoNameFromInsideWorktree(t *testing.T) {
	// Set up a main repo, then add a worktree inside it. Running `jotter
	// project` from the worktree should still return the main repo's name.
	mainRepo := t.TempDir()
	for _, cmdArgs := range [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
		{"git", "worktree", "add", "-b", "feature/x", "wt-feature-x"},
	} {
		c := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		c.Dir = mainRepo
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", cmdArgs, out)
		}
	}

	worktreeDir := filepath.Join(mainRepo, "wt-feature-x")
	cleanHome := t.TempDir()
	cmd := exec.Command(binaryPath, "project")
	cmd.Dir = worktreeDir
	cmd.Env = append(os.Environ(), "HOME="+cleanHome)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("jotter project failed: %v, stderr: %s", err, stderr.String())
	}

	want := filepath.Base(mainRepo)
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Errorf("project = %q, want %q (main repo basename, not worktree dir)", got, want)
	}
}

func TestProject_ErrorsOutsideGitRepo(t *testing.T) {
	// Use the non-git runJotter helper so cwd is not a git repo.
	dir := initDataDir(t)
	_, stderr, code := runJotter(t, dir, "project")
	if code == 0 {
		t.Error("expected non-zero exit code outside a git repo")
	}
	if !strings.Contains(stderr, "not inside a git repo") {
		t.Errorf("stderr missing expected message: %s", stderr)
	}
}

func TestBranch_PrintsCurrentBranch(t *testing.T) {
	stdout, _, code, _ := runJotterFromGitRepo(t, "feature/test-branch", "branch")
	if code != 0 {
		t.Fatalf("exit code %d, stdout: %s", code, stdout)
	}
	if got := strings.TrimSpace(stdout); got != "feature/test-branch" {
		t.Errorf("branch = %q, want feature/test-branch", got)
	}
}

func TestBranch_ErrorsOnDetachedHEAD(t *testing.T) {
	workdir := t.TempDir()
	for _, cmdArgs := range [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "one"},
		{"git", "commit", "--allow-empty", "-m", "two"},
	} {
		c := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		c.Dir = workdir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", cmdArgs, out)
		}
	}
	// Detach HEAD by checking out the previous commit's SHA.
	c := exec.Command("git", "rev-parse", "HEAD~1")
	c.Dir = workdir
	sha, err := c.Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	c = exec.Command("git", "checkout", strings.TrimSpace(string(sha)))
	c.Dir = workdir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("checkout detached: %s", out)
	}

	cleanHome := t.TempDir()
	cmd := exec.Command(binaryPath, "branch")
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "HOME="+cleanHome)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit on detached HEAD, stderr: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "detached HEAD") {
		t.Errorf("stderr missing expected message: %s", stderr.String())
	}
}
