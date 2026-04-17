package internal

import (
	"fmt"
	"path/filepath"
	"sort"
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

// CollectPaths returns JSONL file paths scoped by project and/or branch.
// Empty project or branch means "all". Non-empty values are validated.
func CollectPaths(dataDir, project, branch string) ([]string, error) {
	if project != "" {
		if err := ValidatePathComponent("project", project); err != nil {
			return nil, err
		}
	}
	if branch != "" {
		if err := ValidatePathComponent("branch", SanitiseBranch(branch)); err != nil {
			return nil, err
		}
	}

	logsDir := filepath.Join(dataDir, "logs")

	if project != "" && branch != "" {
		path, err := JSONLPath(dataDir, project, branch)
		if err != nil {
			return nil, err
		}
		return []string{path}, nil
	}

	if project != "" {
		pattern := filepath.Join(logsDir, project, "*.jsonl")
		matches, _ := filepath.Glob(pattern)
		sort.Strings(matches)
		return matches, nil
	}

	pattern := filepath.Join(logsDir, "*", "*.jsonl")
	matches, _ := filepath.Glob(pattern)
	sort.Strings(matches)
	return matches, nil
}
