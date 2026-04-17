package setup

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sebjacobs/jotter/internal"
)

// DefaultSteps returns the seven-step wizard sequence in execution order.
func DefaultSteps() []Step {
	return []Step{
		&claudeStep{},
		&dataDirStep{},
		&remoteStep{},
		&configStep{},
		&skillsStep{},
		&permissionStep{},
		&smokeStep{},
	}
}

// --- step 1: Claude Code presence ---

type claudeStep struct{}

func (claudeStep) Name() string { return "Claude Code" }

func (claudeStep) Detect(ctx *Context) (State, error) {
	info, err := os.Stat(filepath.Join(ctx.Home, ".claude"))
	switch {
	case os.IsNotExist(err):
		return NotApplicable, nil
	case err != nil:
		return 0, err
	case !info.IsDir():
		return NotApplicable, nil
	}
	return AlreadyDone, nil // Claude Code is installed; nothing to do for this step
}

func (claudeStep) Run(_ *Context) (Result, error) {
	return Result{Status: StatusOK, Message: "Claude Code detected"}, nil
}

// --- step 2: data directory ---

type dataDirStep struct{}

func (dataDirStep) Name() string { return "data directory" }

// Detect reads ~/.jotter (if present) and pre-populates Answers.DataDir so
// Run's prompt shows the existing path as the default. Always returns
// NeedsRun — the user may legitimately want to change the data dir, and the
// prompt gives them that opportunity without forcing them to edit ~/.jotter
// by hand. Accepting the default is a no-op: Run only returns StatusUpdated
// when the chosen path differs from what was already configured.
func (dataDirStep) Detect(ctx *Context) (State, error) {
	configPath := filepath.Join(ctx.Home, ".jotter")
	if _, err := os.Stat(configPath); err != nil {
		return NeedsRun, nil
	}
	cfg, err := internal.LoadConfig(configPath)
	if err == nil {
		ctx.Answers.DataDir = cfg.DataDir
	}
	return NeedsRun, nil
}

func (dataDirStep) Run(ctx *Context) (Result, error) {
	defaultPath := ctx.Answers.DataDir
	if defaultPath == "" {
		defaultPath = filepath.Join(ctx.Home, "session-logs-data")
	}
	path, err := ctx.Prompter.Input("Where should session logs live?", defaultPath)
	if err != nil {
		return Result{}, err
	}
	path = expandHome(path, ctx.Home)
	ctx.Answers.DataDir = path

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return Result{}, fmt.Errorf("creating %s: %w", path, err)
		}
		if err := runGit(path, "init"); err != nil {
			return Result{}, fmt.Errorf("git init %s: %w", path, err)
		}
		return Result{Status: StatusUpdated, Message: fmt.Sprintf("created and initialised %s", path)}, nil
	}
	if err != nil {
		return Result{}, err
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("%s exists and is not a directory", path)
	}

	if _, err := os.Stat(filepath.Join(path, ".git")); os.IsNotExist(err) {
		confirm, err := ctx.Prompter.Confirm(fmt.Sprintf("%s isn't a git repo — run 'git init' there?", path), true)
		if err != nil {
			return Result{}, err
		}
		if !confirm {
			return Result{}, fmt.Errorf("data dir must be a git repository")
		}
		if err := runGit(path, "init"); err != nil {
			return Result{}, fmt.Errorf("git init %s: %w", path, err)
		}
		return Result{Status: StatusUpdated, Message: fmt.Sprintf("initialised %s as a git repo", path)}, nil
	}
	return Result{Status: StatusOK, Message: fmt.Sprintf("using existing git repo at %s", path)}, nil
}

// --- step 3: git remote ---

type remoteStep struct{}

func (remoteStep) Name() string { return "git remote" }

// Detect always returns NeedsRun. Run prompts with the existing remote (if
// any) as the default — accepting the default is a no-op because Run only
// returns StatusUpdated when the URL actually changes. This matches the
// "show current value, let the user edit it" pattern the wizard applies
// everywhere else.
func (remoteStep) Detect(_ *Context) (State, error) { return NeedsRun, nil }

func (remoteStep) Run(ctx *Context) (Result, error) {
	path := ctx.Answers.DataDir
	existing, _ := exec.Command("git", "-C", path, "remote", "get-url", "origin").Output()
	existingURL := strings.TrimSpace(string(existing))

	url, err := ctx.Prompter.Input("Git remote URL for the data repo (blank to skip)", existingURL)
	if err != nil {
		return Result{}, err
	}
	ctx.Answers.RemoteURL = url

	if url == "" {
		return Result{Status: StatusSkipped, Message: "no remote configured (finish entries will not push)"}, nil
	}

	switch {
	case existingURL == "":
		if err := runGit(path, "remote", "add", "origin", url); err != nil {
			return Result{}, fmt.Errorf("adding remote: %w", err)
		}
		return Result{Status: StatusUpdated, Message: fmt.Sprintf("remote origin set to %s", url)}, nil
	case existingURL != url:
		if err := runGit(path, "remote", "set-url", "origin", url); err != nil {
			return Result{}, fmt.Errorf("updating remote: %w", err)
		}
		return Result{Status: StatusUpdated, Message: fmt.Sprintf("remote origin updated to %s", url)}, nil
	default:
		return Result{Status: StatusOK, Message: fmt.Sprintf("remote origin already set to %s", url)}, nil
	}
}

// --- step 4: .jotter config ---

type configStep struct{}

func (configStep) Name() string { return ".jotter config" }

func (configStep) Detect(_ *Context) (State, error) { return NeedsRun, nil }

func (configStep) Run(ctx *Context) (Result, error) {
	path := filepath.Join(ctx.Home, ".jotter")
	desired := fmt.Sprintf("data_dir = %q\n", ctx.Answers.DataDir)

	existing, err := os.ReadFile(path)
	switch {
	case os.IsNotExist(err):
		// fall through to write
	case err != nil:
		return Result{}, err
	default:
		if string(existing) == desired {
			return Result{Status: StatusSkipped, Message: "~/.jotter already up to date"}, nil
		}
		confirm, err := ctx.Prompter.Confirm(fmt.Sprintf("%s already exists — overwrite?", path), false)
		if err != nil {
			return Result{}, err
		}
		if !confirm {
			return Result{Status: StatusSkipped, Message: "kept existing ~/.jotter"}, nil
		}
	}

	if err := os.WriteFile(path, []byte(desired), 0o644); err != nil {
		return Result{}, err
	}
	return Result{Status: StatusUpdated, Message: fmt.Sprintf("wrote %s pointing at %s", path, ctx.Answers.DataDir)}, nil
}

// --- step 5: skills install ---

type skillsStep struct{}

func (skillsStep) Name() string { return "skills" }

func (skillsStep) Detect(_ *Context) (State, error) { return NeedsRun, nil }

func (skillsStep) Run(ctx *Context) (Result, error) {
	skillsRoot := filepath.Join(ctx.Home, ".claude", "skills")
	if err := os.MkdirAll(skillsRoot, 0o755); err != nil {
		return Result{}, err
	}

	var (
		copied  int
		updated int
	)
	root := ctx.skillsRoot()
	err := fs.WalkDir(ctx.SkillsFS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(path, root+"/")
		if rel == root || rel == "" || path == root {
			return nil
		}
		dest := filepath.Join(skillsRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, err := fs.ReadFile(ctx.SkillsFS, path)
		if err != nil {
			return err
		}
		existing, readErr := os.ReadFile(dest)
		if readErr == nil && string(existing) == string(data) {
			return nil // already up to date
		}
		if readErr == nil {
			updated++
		} else {
			copied++
		}
		return os.WriteFile(dest, data, 0o644)
	})
	if err != nil {
		return Result{}, err
	}

	switch {
	case copied == 0 && updated == 0:
		return Result{Status: StatusSkipped, Message: "all skills already up to date"}, nil
	case updated == 0:
		return Result{Status: StatusOK, Message: fmt.Sprintf("installed %d skills to %s", copied, skillsRoot)}, nil
	default:
		return Result{Status: StatusUpdated, Message: fmt.Sprintf("installed %d, updated %d in %s", copied, updated, skillsRoot)}, nil
	}
}

// --- step 6: Claude Code permission ---

type permissionStep struct{}

func (permissionStep) Name() string { return "Claude permission" }

func (permissionStep) Detect(_ *Context) (State, error) { return NeedsRun, nil }

func (permissionStep) Run(ctx *Context) (Result, error) {
	path := filepath.Join(ctx.Home, ".claude", "settings.json")
	changed, err := MergePermission(path, "Bash(jotter:*)")
	if err != nil {
		return Result{}, err
	}
	if !changed {
		return Result{Status: StatusSkipped, Message: "Bash(jotter:*) already allowed"}, nil
	}
	return Result{Status: StatusUpdated, Message: fmt.Sprintf("added Bash(jotter:*) to %s", path)}, nil
}

// --- step 7: smoke test ---

type smokeStep struct{}

func (smokeStep) Name() string { return "smoke test" }

// Detect skips the smoke test if no prior step made a change. Writing (and
// then cleaning up) an entry in the data repo produces two commits every
// time — tolerable on a fresh install, but cruft on a no-op re-run.
func (smokeStep) Detect(ctx *Context) (State, error) {
	if !ctx.Changed {
		return AlreadyDone, nil
	}
	return NeedsRun, nil
}

func (smokeStep) Run(ctx *Context) (Result, error) {
	// Invoke the jotter binary itself rather than calling internal packages
	// directly — tests the end-to-end path including config resolution.
	exe, err := os.Executable()
	if err != nil {
		return Result{}, err
	}

	const smokeProject = "jotter-setup"
	const smokeBranch = "smoke"

	writeArgs := []string{"write", "--project", smokeProject, "--branch", smokeBranch, "--type", "note", "--content", "jotter setup smoke test (cleaned up)"}
	if out, err := exec.Command(exe, writeArgs...).CombinedOutput(); err != nil {
		return Result{}, fmt.Errorf("jotter write failed: %w\n%s", err, string(out))
	}

	tailArgs := []string{"tail", "--project", smokeProject, "--branch", smokeBranch, "--limit", "1"}
	out, err := exec.Command(exe, tailArgs...).CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("jotter tail failed: %w\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "smoke test") {
		return Result{}, fmt.Errorf("smoke-test entry not found in tail output:\n%s", string(out))
	}

	// Clean up the smoke test artefact — both the .jsonl file and the parent
	// project dir (if empty), then commit the removal. Users shouldn't see a
	// phantom "jotter-setup" project in `jotter ls` forever.
	if cleanupErr := cleanupSmokeArtefacts(ctx.Answers.DataDir, smokeProject, smokeBranch); cleanupErr != nil {
		// Don't fail the whole wizard on a cleanup error — the write+read
		// succeeded, which is what the step is really verifying. Surface
		// the cleanup issue in the result message instead.
		return Result{Status: StatusOK, Message: fmt.Sprintf("wrote and read back a test entry (cleanup warning: %v)", cleanupErr)}, nil
	}
	return Result{Status: StatusOK, Message: "wrote and read back a test entry (and cleaned up)"}, nil
}

// cleanupSmokeArtefacts removes the smoke-test .jsonl and parent project dir
// from the data repo, then commits the removal. Separate from the Run body
// so the step stays readable and the cleanup is isolated for testing.
func cleanupSmokeArtefacts(dataDir, project, branch string) error {
	projectDir := filepath.Join(dataDir, "logs", project)
	entryFile := filepath.Join(projectDir, branch+".jsonl")

	if err := os.Remove(entryFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %s: %w", entryFile, err)
	}
	// Remove parent dir if empty — os.Remove returns an error on non-empty
	// dirs, which we treat as "user has other stuff here, leave it alone".
	_ = os.Remove(projectDir)

	// Commit the removal so the data repo stays consistent with the
	// filesystem. Use -A to pick up whatever changed (the delete, plus the
	// empty-dir removal if git tracks it).
	if err := runGit(dataDir, "add", "-A"); err != nil {
		return fmt.Errorf("staging cleanup: %w", err)
	}
	// --allow-empty covers the edge case where git had already pruned the
	// file (e.g. previous setup run left things in a consistent state).
	if err := runGit(dataDir, "commit", "--allow-empty", "-m", "jotter setup: clean up smoke-test artefacts"); err != nil {
		return fmt.Errorf("committing cleanup: %w", err)
	}
	return nil
}

// --- helpers ---

func expandHome(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		return home
	}
	return path
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, string(out))
	}
	return nil
}
