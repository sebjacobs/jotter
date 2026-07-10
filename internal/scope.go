package internal

import (
	"path/filepath"
	"sort"
)

// Scope selects which branch logfiles a query command operates on. It replaces
// the earlier convention of passing empty project/branch strings to mean "all":
// each variant states its breadth explicitly, and BranchScope's named fields
// make a transposed project/branch impossible to express.
type Scope interface {
	// Paths returns the JSONL logfiles in scope under dataDir, sorted. A scope
	// that matches nothing returns an empty slice, not an error.
	Paths(dataDir string) ([]string, error)
}

// AllScope is every branch of every project.
type AllScope struct{}

// ProjectScope is every branch of one project.
type ProjectScope struct{ Project string }

// BranchScope is a single branch of one project.
type BranchScope struct {
	Project string
	Branch  string
}

func (AllScope) Paths(dataDir string) ([]string, error) {
	return globPaths(filepath.Join(dataDir, "logs", "*", "*.jsonl")), nil
}

func (s ProjectScope) Paths(dataDir string) ([]string, error) {
	if err := ValidatePathComponent("project", s.Project); err != nil {
		return nil, err
	}
	return globPaths(filepath.Join(dataDir, "logs", s.Project, "*.jsonl")), nil
}

func (s BranchScope) Paths(dataDir string) ([]string, error) {
	path, err := JSONLPath(dataDir, s.Project, s.Branch)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}

func globPaths(pattern string) []string {
	matches, _ := filepath.Glob(pattern)
	sort.Strings(matches)
	return matches
}
