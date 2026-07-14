package cmd

import (
	"path/filepath"

	"github.com/sebjacobs/jotter/internal"
)

// entrySHAs returns the short commit id for each of a log file's entries, in
// file order, so callers can render an entry alongside the commit that created
// it. path is the absolute JSONL path; entryCount is how many entries
// ReadEntries returned for it.
//
// It returns nil — and callers degrade to no ids rather than fail — whenever
// the ids can't be trusted: blame fails (not a git repo, untracked file), or
// the id count doesn't match entryCount. A mismatch means a commit touched the
// file without appending exactly one entry (an old-format log, a hand-edit),
// so positional alignment would attach wrong ids; better to show none.
func entrySHAs(dataDir, path string, entryCount int) []string {
	rel, err := filepath.Rel(dataDir, path)
	if err != nil {
		return nil
	}
	shas, err := internal.GitLineSHAs(dataDir, rel)
	if err != nil || len(shas) != entryCount {
		return nil
	}
	return shas
}
