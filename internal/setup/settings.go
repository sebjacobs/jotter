package setup

import (
	"encoding/json"
	"fmt"
	"os"
)

// MergePermission adds `entry` to the permissions.allow list in a Claude Code
// settings.json file, preserving all other content. Returns true if the file
// was modified (false if the entry was already present).
//
// If path doesn't exist, a minimal settings.json containing just the
// permission is written.
func MergePermission(path, entry string) (changed bool, err error) {
	var settings map[string]any

	data, err := os.ReadFile(path)
	switch {
	case os.IsNotExist(err):
		settings = map[string]any{}
	case err != nil:
		return false, fmt.Errorf("reading %s: %w", path, err)
	default:
		if len(data) == 0 {
			settings = map[string]any{}
		} else if err := json.Unmarshal(data, &settings); err != nil {
			return false, fmt.Errorf("parsing %s: %w", path, err)
		}
	}

	changed = addToAllowList(settings, entry)
	if !changed {
		return false, nil
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshalling settings: %w", err)
	}
	// Preserve trailing newline — conventional for JSON files edited by humans.
	out = append(out, '\n')

	if err := os.WriteFile(path, out, 0o644); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}
	return true, nil
}

// addToAllowList mutates settings to contain permissions.allow with entry.
// Returns true if the structure changed; false if entry was already present.
func addToAllowList(settings map[string]any, entry string) bool {
	permissions, _ := settings["permissions"].(map[string]any)
	if permissions == nil {
		permissions = map[string]any{}
		settings["permissions"] = permissions
	}

	allow, _ := permissions["allow"].([]any)
	for _, v := range allow {
		if s, ok := v.(string); ok && s == entry {
			return false
		}
	}

	allow = append(allow, entry)
	permissions["allow"] = allow
	return true
}
