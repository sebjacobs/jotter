package internal

import (
	"crypto/rand"
	"encoding/hex"
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
