package internal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sidecarSuffix is appended to a branch logfile path to name its id sidecar.
const sidecarSuffix = ".id"

const anchorVar = "jotter-id"

// AnchorConfigKey returns the git config key holding a branch's stable jotter
// id, e.g. branch.feature/foo.jotter-id. The raw (un-sanitised) branch name is
// used deliberately — git config subsections accept "/".
func AnchorConfigKey(branch string) string {
	return "branch." + branch + "." + anchorVar
}

// NewID returns a fresh 32-hex-character random id. crypto/rand.Read does not
// fail on supported platforms, so the error is intentionally discarded.
func NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// SidecarPath returns the path to a branch logfile's id sidecar:
// logs/<project>/<sanitised-branch>.jsonl.id .
func SidecarPath(dataDir, project, branch string) (string, error) {
	jsonl, err := JSONLPath(dataDir, project, branch)
	if err != nil {
		return "", err
	}
	return jsonl + sidecarSuffix, nil
}

// ReadSidecar returns the id stored in a sidecar file, or "" if it is absent.
func ReadSidecar(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// WriteSidecar writes id to a sidecar file as a single line.
func WriteSidecar(path, id string) error {
	return os.WriteFile(path, []byte(id+"\n"), 0o644)
}

// FindLogByID scans logs/<project>/*.jsonl.id and returns the sanitised branch
// basename (the part before ".jsonl.id") whose sidecar holds id, or "" if none
// matches.
func FindLogByID(dataDir, project, id string) (string, error) {
	if err := ValidatePathComponent("project", project); err != nil {
		return "", err
	}
	matches, _ := filepath.Glob(filepath.Join(dataDir, "logs", project, "*.jsonl"+sidecarSuffix))
	for _, m := range matches {
		got, err := ReadSidecar(m)
		if err != nil {
			return "", err
		}
		if got == id {
			return strings.TrimSuffix(filepath.Base(m), ".jsonl"+sidecarSuffix), nil
		}
	}
	return "", nil
}

// ReconcileBranch ensures the logfile for (project, branch) is named for the
// current branch, following a git rename if one happened, and returns the path
// the caller should append the entry to.
//
// It anchors the branch's identity in cwd's repo config on first sight, and on
// later writes uses that anchor to detect a rename and git-mv the stale logfile
// (and its sidecar) into place. It is best-effort: a non-nil warn means tracking
// was skipped this time, and logPath falls back to the current-name file so the
// write still lands. Only a genuine path-construction error is returned as a
// hard error (empty logPath).
func ReconcileBranch(dataDir, cwd, project, branch string) (logPath string, warn error) {
	fallback, err := JSONLPath(dataDir, project, branch)
	if err != nil {
		return "", err
	}
	sidecar, err := SidecarPath(dataDir, project, branch)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(fallback), 0o755); err != nil {
		return fallback, err
	}

	id, err := GitConfigGet(cwd, AnchorConfigKey(branch))
	if err != nil {
		return fallback, err
	}

	if id == "" {
		// Not yet anchored. Adopt an existing sidecar's id if one is somehow
		// already present (e.g. branch deleted then recreated under the same
		// name), otherwise mint a fresh one.
		existing, _ := ReadSidecar(sidecar)
		if existing == "" {
			existing = NewID()
		}
		if err := GitConfigSet(cwd, AnchorConfigKey(branch), existing); err != nil {
			return fallback, err
		}
		if err := WriteSidecar(sidecar, existing); err != nil {
			return fallback, err
		}
		return fallback, nil
	}

	loc, err := FindLogByID(dataDir, project, id)
	if err != nil {
		return fallback, err
	}
	sanitised := SanitiseBranch(branch)
	switch loc {
	case sanitised:
		return fallback, nil
	case "":
		// Anchored, but no logfile carries the id yet (first write, or the file
		// was removed). Stamp the current-name file.
		if err := WriteSidecar(sidecar, id); err != nil {
			return fallback, err
		}
		return fallback, nil
	default:
		// loc holds the stale (sanitised) name — a rename happened.
		if err := MoveBranchLogs(dataDir, project, UnsanitiseBranch(loc), branch); err != nil {
			return fallback, err
		}
		return fallback, nil
	}
}

// MoveBranchLogs git-mv's a branch's logfile and its id sidecar from old to new
// within the data repo and commits the move. Shared by ReconcileBranch and the
// `jotter branch mv` command. Refuses when the destination logfile already
// exists, mirroring `jotter mv`'s refuse-to-overwrite guard.
func MoveBranchLogs(dataDir, project, old, new string) error {
	if old == new {
		return nil
	}
	for _, c := range []struct{ kind, val string }{
		{"project", project},
		{"branch", SanitiseBranch(old)},
		{"branch", SanitiseBranch(new)},
	} {
		if err := ValidatePathComponent(c.kind, c.val); err != nil {
			return err
		}
	}

	oldRel := filepath.Join("logs", project, SanitiseBranch(old)+".jsonl")
	newRel := filepath.Join("logs", project, SanitiseBranch(new)+".jsonl")

	if _, err := os.Stat(filepath.Join(dataDir, oldRel)); err != nil {
		return fmt.Errorf("no logs for branch %q in project %q", old, project)
	}
	if _, err := os.Stat(filepath.Join(dataDir, newRel)); err == nil {
		return fmt.Errorf("branch %q already has logs — refusing to overwrite", new)
	}

	if err := GitMove(dataDir, oldRel, newRel); err != nil {
		return fmt.Errorf("renaming logs: %w", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, oldRel+sidecarSuffix)); err == nil {
		if err := GitMove(dataDir, oldRel+sidecarSuffix, newRel+sidecarSuffix); err != nil {
			return fmt.Errorf("renaming sidecar: %w", err)
		}
	}
	return GitCommitStaged(dataDir, fmt.Sprintf("rename: %s -> %s", oldRel, newRel))
}
