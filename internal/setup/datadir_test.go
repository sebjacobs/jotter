package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestDataDirStepDetectReadsExistingConfig verifies that Detect pre-populates
// Answers.DataDir from an existing ~/.jotter so subsequent prompts show the
// right default rather than clobbering the user's config with the generic
// ~/session-logs-data default.
func TestDataDirStepDetectReadsExistingConfig(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{})

	// Existing data dir, git-initialised.
	existingData := filepath.Join(ctx.Home, "my-logs")
	if err := os.MkdirAll(filepath.Join(existingData, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Existing ~/.jotter pointing at it.
	configPath := filepath.Join(ctx.Home, ".jotter")
	if err := os.WriteFile(configPath, []byte(`data_dir = "`+existingData+`"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := dataDirStep{}.Detect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state != AlreadyDone {
		t.Errorf("state = %v, want AlreadyDone (data dir exists + is git repo)", state)
	}
	if ctx.Answers.DataDir != existingData {
		t.Errorf("Answers.DataDir = %q, want %q (pre-populated from ~/.jotter)", ctx.Answers.DataDir, existingData)
	}
}

// TestDataDirStepDetectPrePopulatesOnInvalidDataDir covers the subtle case the
// user ran into: ~/.jotter exists but points at a path that's missing or not
// a git repo. We must still pre-populate Answers.DataDir so the Run prompt
// shows the existing path as the default, rather than silently defaulting to
// ~/session-logs-data (which would clobber the user's real config on a
// naive hit-enter accept).
func TestDataDirStepDetectPrePopulatesOnMissingDataDir(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{})

	existingData := filepath.Join(ctx.Home, "does-not-exist")
	configPath := filepath.Join(ctx.Home, ".jotter")
	if err := os.WriteFile(configPath, []byte(`data_dir = "`+existingData+`"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := dataDirStep{}.Detect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state != NeedsRun {
		t.Errorf("state = %v, want NeedsRun (data dir doesn't exist)", state)
	}
	if ctx.Answers.DataDir != existingData {
		t.Errorf("Answers.DataDir = %q, want %q (pre-populated so Run prompt uses it as default)", ctx.Answers.DataDir, existingData)
	}
}

func TestDataDirStepDetectNoExistingConfig(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{})

	state, err := dataDirStep{}.Detect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state != NeedsRun {
		t.Errorf("state = %v, want NeedsRun (no ~/.jotter present)", state)
	}
	if ctx.Answers.DataDir != "" {
		t.Errorf("Answers.DataDir = %q, want empty (nothing to pre-populate)", ctx.Answers.DataDir)
	}
}

// TestDataDirStepRunUsesPrePopulatedDefault confirms the prompt default
// respects Answers.DataDir when Detect pre-populated it — the fix for the
// clobber bug.
func TestDataDirStepRunUsesPrePopulatedDefault(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{inputs: []string{"__DEFAULT__"}}) // accept default

	existingData := t.TempDir()
	if err := exec.Command("git", "-C", existingData, "init").Run(); err != nil {
		t.Fatal(err)
	}
	ctx.Answers.DataDir = existingData // as if Detect pre-populated it

	result, err := dataDirStep{}.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Answers.DataDir != existingData {
		t.Errorf("Answers.DataDir was clobbered: got %q, want %q", ctx.Answers.DataDir, existingData)
	}
	if result.Status != StatusOK {
		t.Errorf("result.Status = %v, want StatusOK (existing git repo)", result.Status)
	}
}
