package internal

import (
	"path/filepath"
	"sort"
	"strings"
)

// SanitiseBranch replaces / with + for use as a filename.
func SanitiseBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "+")
}

// JSONLPath returns the path to a branch's JSONL log file.
func JSONLPath(dataDir, project, branch string) string {
	return filepath.Join(dataDir, "logs", project, SanitiseBranch(branch)+".jsonl")
}

// CollectPaths returns JSONL file paths scoped by project and/or branch.
func CollectPaths(dataDir, project, branch string) []string {
	logsDir := filepath.Join(dataDir, "logs")

	if project != "" && branch != "" {
		return []string{JSONLPath(dataDir, project, branch)}
	}

	if project != "" {
		pattern := filepath.Join(logsDir, project, "*.jsonl")
		matches, _ := filepath.Glob(pattern)
		sort.Strings(matches)
		return matches
	}

	pattern := filepath.Join(logsDir, "*", "*.jsonl")
	matches, _ := filepath.Glob(pattern)
	sort.Strings(matches)
	return matches
}
