package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"
)

// Entry represents a single session log entry.
type Entry struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Next      string `json:"next,omitempty"`
}

// ValidEntryTypes lists all allowed entry types, in canonical order.
var ValidEntryTypes = []string{"start", "checkpoint", "note", "break", "finish"}

// IsValidEntryType checks whether a type string is one of the allowed entry types.
func IsValidEntryType(t string) bool {
	return slices.Contains(ValidEntryTypes, t)
}

// ReadEntries reads all entries from a JSONL file.
// Returns an empty slice if the file doesn't exist.
func ReadEntries(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("invalid JSONL line: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

// MarshalJSONL serialises an entry as a single JSON line with Python-compatible
// spacing (spaces after : and ,) to match json.dumps default output.
func MarshalJSONL(e Entry) ([]byte, error) {
	return marshalPythonCompat(e)
}

// marshalPythonCompat builds JSON matching Python's json.dumps default separators.
func marshalPythonCompat(e Entry) ([]byte, error) {
	parts := []string{
		fmt.Sprintf(`"timestamp": %s`, quote(e.Timestamp)),
		fmt.Sprintf(`"type": %s`, quote(e.Type)),
		fmt.Sprintf(`"content": %s`, quote(e.Content)),
	}
	if e.Next != "" {
		parts = append(parts, fmt.Sprintf(`"next": %s`, quote(e.Next)))
	}
	return []byte("{" + strings.Join(parts, ", ") + "}"), nil
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// FormatEntry renders an entry as markdown with color.
// Colors are automatically disabled when output is not a terminal.
func FormatEntry(e Entry) string {
	t, _ := time.Parse(time.RFC3339, e.Timestamp)
	if t.IsZero() {
		t, _ = time.Parse("2006-01-02T15:04:05", e.Timestamp)
	}
	heading := fmt.Sprintf("## %s | %s", Dim(t.Format("2006-01-02 15:04")), ColorType(e.Type))

	lines := []string{heading, "", e.Content}
	if e.Next != "" {
		lines = append(lines, "", fmt.Sprintf("%s %s", Bold("Next:"), e.Next))
	}
	return strings.Join(lines, "\n")
}
