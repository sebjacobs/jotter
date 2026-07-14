package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SanitiseBranch replaces / with + for use as a filename.
func SanitiseBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "+")
}

// UnsanitiseBranch reverses SanitiseBranch, restoring / from +.
func UnsanitiseBranch(branch string) string {
	return strings.ReplaceAll(branch, "+", "/")
}

// ValidatePathComponent rejects values that would escape the intended directory
// when joined into a filesystem path. kind is used only in the error message.
//
// A component is rejected if it is empty, equal to "." or "..", contains "..",
// contains a path separator or backslash, contains a null byte, or begins with
// "." (which would conflict with hidden files and "." / "..").
func ValidatePathComponent(kind, value string) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", kind)
	}
	if value == "." || value == ".." {
		return fmt.Errorf("%s %q is not allowed", kind, value)
	}
	if strings.HasPrefix(value, ".") {
		return fmt.Errorf("%s must not start with '.'", kind)
	}
	if strings.Contains(value, "..") {
		return fmt.Errorf("%s must not contain '..'", kind)
	}
	if strings.ContainsAny(value, `/\`) {
		return fmt.Errorf("%s must not contain path separators", kind)
	}
	if strings.ContainsRune(value, 0) {
		return fmt.Errorf("%s must not contain null bytes", kind)
	}
	return nil
}

// RemoveEntryLine removes the first log line byte-equal to line from whichever
// current logfile under logs/ contains it, rewrites that file, and returns the
// file's path relative to dataDir so the caller can commit the removal. The
// tree it produces is exactly what a clean git revert of the entry's commit
// would — the appended line, gone — but locating the line by content follows
// any branch rename and leaves the branch-id sidecar untouched.
//
// Returns an error if no current logfile contains the line (already removed, or
// on a branch whose logs have since moved out of this store).
func RemoveEntryLine(dataDir, line string) (string, error) {
	for _, path := range globPaths(filepath.Join(dataDir, "logs", "*", "*.jsonl")) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for i, l := range lines {
			if l != line {
				continue
			}
			kept := append(lines[:i:i], lines[i+1:]...)
			body := strings.Join(kept, "\n")
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				return "", fmt.Errorf("rewriting %s: %w", path, err)
			}
			rel, _ := filepath.Rel(dataDir, path)
			return rel, nil
		}
	}
	return "", fmt.Errorf("entry not found in any current log — already removed?")
}

// JSONLPath returns the path to a branch's JSONL log file, after validating
// that project and branch cannot escape the logs directory.
func JSONLPath(dataDir, project, branch string) (string, error) {
	if err := ValidatePathComponent("project", project); err != nil {
		return "", err
	}
	if err := ValidatePathComponent("branch", SanitiseBranch(branch)); err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "logs", project, SanitiseBranch(branch)+".jsonl"), nil
}
