package cmd_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
