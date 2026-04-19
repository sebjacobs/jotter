package setup

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubPrompter returns canned answers — zero-th answer for the first call,
// first for the second, etc. Unused answers are fine; missing answers panic.
type stubPrompter struct {
	inputs   []string
	confirms []bool
	inputIdx int
	confIdx  int
}

func (s *stubPrompter) Input(_, defaultValue string) (string, error) {
	if s.inputIdx >= len(s.inputs) {
		return defaultValue, nil
	}
	v := s.inputs[s.inputIdx]
	s.inputIdx++
	if v == "__DEFAULT__" {
		return defaultValue, nil
	}
	return v, nil
}

func (s *stubPrompter) Confirm(_ string, defaultYes bool) (bool, error) {
	if s.confIdx >= len(s.confirms) {
		return defaultYes, nil
	}
	v := s.confirms[s.confIdx]
	s.confIdx++
	return v, nil
}

func newStepCtx(t *testing.T, prompter Prompter) *Context {
	t.Helper()
	home := t.TempDir()
	return &Context{
		Home:     home,
		Answers:  &Answers{},
		Prompter: prompter,
	}
}

func TestClaudeStepNotApplicableWhenMissing(t *testing.T) {
	ctx := newStepCtx(t, nil)
	s := claudeStep{}
	state, err := s.Detect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state != NotApplicable {
		t.Errorf("expected NotApplicable, got %v", state)
	}
}

func TestClaudeStepAlreadyDoneWhenPresent(t *testing.T) {
	ctx := newStepCtx(t, nil)
	if err := os.MkdirAll(filepath.Join(ctx.Home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	state, err := claudeStep{}.Detect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state != AlreadyDone {
		t.Errorf("expected AlreadyDone, got %v", state)
	}
}

func TestConfigStepWritesNewFile(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{})
	ctx.Answers.DataDir = "/tmp/logs"

	result, err := configStep{}.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusUpdated {
		t.Errorf("status = %v, want StatusUpdated", result.Status)
	}

	data, err := os.ReadFile(filepath.Join(ctx.Home, ".jotter"))
	if err != nil {
		t.Fatal(err)
	}
	want := `data_dir = "/tmp/logs"` + "\n"
	if string(data) != want {
		t.Errorf(".jotter = %q, want %q", string(data), want)
	}
}

func TestConfigStepSkipsWhenIdentical(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{})
	ctx.Answers.DataDir = "/tmp/logs"

	// First run writes.
	if _, err := (configStep{}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	// Second run should skip.
	result, err := (configStep{}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusSkipped {
		t.Errorf("status = %v, want StatusSkipped", result.Status)
	}
}

func TestConfigStepPromptsBeforeOverwrite(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{confirms: []bool{false}}) // decline overwrite
	path := filepath.Join(ctx.Home, ".jotter")
	if err := os.WriteFile(path, []byte(`data_dir = "/other"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx.Answers.DataDir = "/new/path"

	result, err := configStep{}.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusSkipped {
		t.Errorf("status = %v, want StatusSkipped (declined overwrite)", result.Status)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "/other") {
		t.Errorf("existing content was clobbered despite decline; got %q", string(data))
	}
}

//go:embed all:testdata/integrations/claude
var testSkillsFS embed.FS

func TestSkillsStepCopiesAllFiles(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{})
	ctx.SkillsFS = testSkillsFS
	ctx.SkillsRoot = "testdata/integrations/claude"

	result, err := (skillsStep{}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusOK {
		t.Errorf("status = %v, want StatusOK (fresh install)", result.Status)
	}

	installed := filepath.Join(ctx.Home, ".claude", "skills")
	for _, rel := range []string{"alpha/SKILL.md", "beta/SKILL.md"} {
		got, err := os.ReadFile(filepath.Join(installed, rel))
		if err != nil {
			t.Errorf("skill %s not installed: %v", rel, err)
			continue
		}
		if !strings.Contains(string(got), "test fixture") {
			t.Errorf("skill %s content unexpected: %q", rel, string(got))
		}
	}
}

func TestSkillsStepIdempotent(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{})
	ctx.SkillsFS = testSkillsFS
	ctx.SkillsRoot = "testdata/integrations/claude"

	if _, err := (skillsStep{}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	result, err := (skillsStep{}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusSkipped {
		t.Errorf("second run status = %v, want StatusSkipped", result.Status)
	}
}

func TestSkillsStepPromptsBeforeOverwrite(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{confirms: []bool{false}}) // decline overwrite
	ctx.SkillsFS = testSkillsFS
	ctx.SkillsRoot = "testdata/integrations/claude"

	// Seed one skill with local edits that differ from the bundled template.
	installed := filepath.Join(ctx.Home, ".claude", "skills")
	if err := os.MkdirAll(filepath.Join(installed, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	localContent := []byte("# local customisation — do not clobber\n")
	alphaPath := filepath.Join(installed, "alpha", "SKILL.md")
	if err := os.WriteFile(alphaPath, localContent, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := (skillsStep{}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// alpha declined; beta is a fresh install (no prompt), so status is Updated.
	if result.Status != StatusUpdated {
		t.Errorf("status = %v, want StatusUpdated (1 installed + 1 kept)", result.Status)
	}

	got, err := os.ReadFile(alphaPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(localContent) {
		t.Errorf("local content clobbered despite decline; got %q, want %q", string(got), string(localContent))
	}
}

func TestSkillsStepOverwritesOnAccept(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{confirms: []bool{true}}) // accept overwrite
	ctx.SkillsFS = testSkillsFS
	ctx.SkillsRoot = "testdata/integrations/claude"

	installed := filepath.Join(ctx.Home, ".claude", "skills")
	if err := os.MkdirAll(filepath.Join(installed, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	alphaPath := filepath.Join(installed, "alpha", "SKILL.md")
	if err := os.WriteFile(alphaPath, []byte("# stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := (skillsStep{}).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusUpdated {
		t.Errorf("status = %v, want StatusUpdated", result.Status)
	}

	got, err := os.ReadFile(alphaPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "test fixture") {
		t.Errorf("skill not overwritten on accept; got %q", string(got))
	}
}

func TestPermissionStepWrapsMerge(t *testing.T) {
	ctx := newStepCtx(t, &stubPrompter{})
	if err := os.MkdirAll(filepath.Join(ctx.Home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := permissionStep{}.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusUpdated {
		t.Errorf("first run status = %v, want StatusUpdated", result.Status)
	}

	// Re-run: should skip.
	result, err = permissionStep{}.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusSkipped {
		t.Errorf("second run status = %v, want StatusSkipped", result.Status)
	}
}
