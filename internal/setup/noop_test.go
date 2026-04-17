package setup

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestWizardIsNoOpOnHealthyInstall verifies the core guarantee: re-running
// jotter setup on a system where everything is already configured correctly,
// with the user accepting every prompt default, makes no file writes and no
// data-repo commits.
//
// The wizard is chatty by design — every re-run shows the current values as
// prompt defaults so the user can see and edit them. But accepting every
// default must be a no-op end-to-end: ctx.Changed stays false, the smoke
// step skips (no data-repo commits), and no step reports StatusUpdated.
func TestWizardIsNoOpOnHealthyInstall(t *testing.T) {
	ctx := newStepCtx(t, &defaultsPrompter{})

	// ~/.claude/ so claudeStep treats the env as applicable.
	if err := os.MkdirAll(filepath.Join(ctx.Home, ".claude", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Data dir, git-initialised, with an origin remote.
	dataDir := filepath.Join(ctx.Home, "session-logs-data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dataDir, "init").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dataDir, "remote", "add", "origin", "git@example.com:me/logs.git").Run(); err != nil {
		t.Fatal(err)
	}

	// ~/.jotter pointing at the data dir.
	configContent := `data_dir = "` + dataDir + `"` + "\n"
	if err := os.WriteFile(filepath.Join(ctx.Home, ".jotter"), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Skills already installed — copy the embedded test fixture into place
	// so the skills step finds a byte-for-byte match.
	ctx.SkillsFS = testSkillsFS
	ctx.SkillsRoot = "testdata/skills"
	for _, skill := range []string{"alpha", "beta"} {
		dest := filepath.Join(ctx.Home, ".claude", "skills", skill)
		if err := os.MkdirAll(dest, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Populate with the same bytes the embed has.
	copySkillFile := func(name string) {
		src, err := testSkillsFS.ReadFile("testdata/skills/" + name + "/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join(ctx.Home, ".claude", "skills", name, "SKILL.md")
		if err := os.WriteFile(dest, src, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	copySkillFile("alpha")
	copySkillFile("beta")

	// settings.json with the permission already present.
	settingsPath := filepath.Join(ctx.Home, ".claude", "settings.json")
	settingsContent := `{
  "permissions": {
    "allow": [
      "Bash(jotter:*)"
    ]
  }
}
`
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Record the data-repo commit SHA before running the wizard.
	shaBefore := headSHA(t, dataDir)

	var out bytes.Buffer
	ctx.Out = &out
	if err := Run(ctx, DefaultSteps()); err != nil {
		t.Fatalf("Run returned error: %v\noutput:\n%s", err, out.String())
	}

	if ctx.Changed {
		t.Errorf("ctx.Changed = true after no-op run; output:\n%s", out.String())
	}

	shaAfter := headSHA(t, dataDir)
	if shaBefore != shaAfter {
		t.Errorf("data repo HEAD changed during no-op run: %s -> %s", shaBefore, shaAfter)
	}

	// Every step should have produced either ↷ (skipped/already-done) or ✓ (no-op ok).
	// No ✎ (updated) allowed.
	if strings.Contains(out.String(), " ✎ ") {
		t.Errorf("no-op run produced at least one StatusUpdated step; output:\n%s", out.String())
	}
}

// defaultsPrompter accepts every default value — simulates a user hitting
// enter through every prompt. Any step that then writes or commits anything
// is a no-op failure.
type defaultsPrompter struct{}

func (defaultsPrompter) Input(_, defaultValue string) (string, error) {
	return defaultValue, nil
}

func (defaultsPrompter) Confirm(_ string, defaultYes bool) (bool, error) {
	return defaultYes, nil
}

func headSHA(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		// Empty repo — no commits yet. Return a sentinel so the test still
		// catches a change (a new commit would make HEAD resolvable).
		return "<empty>"
	}
	return strings.TrimSpace(string(out))
}
