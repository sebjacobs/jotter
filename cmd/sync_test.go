package cmd_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runJotterEnv runs the jotter binary from workdir with the given extra env
// entries appended (e.g. HOME, JOTTER_STATE_DIR). Unlike runJotter it does not
// fabricate a workdir or .jotter file, so callers can share a state dir and a
// HOME across several invocations — needed to exercise the registry that
// `sync --all` reads.
func runJotterEnv(t *testing.T, workdir string, env []string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run jotter: %v", err)
		}
	}
	return stdout.String(), stderr.String(), code
}

// workdirFor returns a temp working directory holding a .jotter config that
// points at dataDir, so a jotter invocation run from it resolves that repo.
func workdirFor(t *testing.T, dataDir string) string {
	t.Helper()
	wd := t.TempDir()
	body := fmt.Sprintf("data_dir = %q\n", dataDir)
	if err := os.WriteFile(filepath.Join(wd, ".jotter"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return wd
}

func TestSyncAll_PushesRemoteReposAndSkipsRemoteless(t *testing.T) {
	stateDir := t.TempDir()
	home := t.TempDir()
	env := []string{"HOME=" + home, "JOTTER_STATE_DIR=" + stateDir}

	withRemote, bare := initDataDirWithRemote(t)
	noRemote := initDataDir(t)

	for _, dir := range []string{withRemote, noRemote} {
		_, stderr, code := runJotterEnv(t, workdirFor(t, dir), env,
			"write", "--project", "proj", "--branch", "main",
			"--type", "checkpoint", "--content", "Work in "+filepath.Base(dir))
		if code != 0 {
			t.Fatalf("write in %s failed: %s", dir, stderr)
		}
	}

	stdout, stderr, code := runJotterEnv(t, t.TempDir(), env, "sync", "--all")
	if code != 0 {
		t.Fatalf("sync --all exit %d, stdout: %s stderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Synced 1 repo") {
		t.Errorf("expected one repo synced: %s", stdout)
	}
	if !strings.Contains(stdout, "skipped 1 without a remote") {
		t.Errorf("expected one repo skipped: %s", stdout)
	}

	log := git(t, bare, "log", "--oneline")
	if !strings.Contains(log, "session: proj/main checkpoint") {
		t.Errorf("remote-backed repo's commit did not reach the bare remote: %s", log)
	}
}

func TestSyncAll_EmptyRegistry(t *testing.T) {
	stateDir := t.TempDir()
	home := t.TempDir()
	env := []string{"HOME=" + home, "JOTTER_STATE_DIR=" + stateDir}

	stdout, stderr, code := runJotterEnv(t, t.TempDir(), env, "sync", "--all")
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "No data repos registered") {
		t.Errorf("expected empty-registry message: %s", stdout)
	}
}
