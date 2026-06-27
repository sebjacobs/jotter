package internal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// StateDirEnv overrides the jotter state directory. When unset, the state
// directory is ~/.jotter.d. Tests set it to point at an isolated temp dir.
const StateDirEnv = "JOTTER_STATE_DIR"

// stateDirName is the per-user state directory holding the registry and the
// daemon log. It sits alongside the ~/.jotter config file.
const stateDirName = ".jotter.d"

// registryFileName lists the data dirs the background push job should sync,
// one absolute path per line.
const registryFileName = "registry"

// StateDir returns the jotter state directory: $JOTTER_STATE_DIR if set,
// otherwise ~/.jotter.d. Mirrors the home-dir override pattern in config.go.
func StateDir() (string, error) {
	if dir := os.Getenv(StateDirEnv); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home directory: %w", err)
	}
	return filepath.Join(home, stateDirName), nil
}

// RegistryPath returns the path to the data-dir registry file.
func RegistryPath() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, registryFileName), nil
}

// RegisterDataDir records dataDir in the registry so the background push job
// knows to sync it. The path is cleaned to an absolute form and only appended
// when absent. Concurrent writers may race to append a duplicate line, which is
// harmless: RegisteredDataDirs dedupes on read.
func RegisterDataDir(dataDir string) error {
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		return err
	}

	existing, err := RegisteredDataDirs()
	if err != nil {
		return err
	}
	if slices.Contains(existing, abs) {
		return nil
	}

	path, err := RegistryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening registry: %w", err)
	}
	_, writeErr := fmt.Fprintln(f, abs)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// RegisteredDataDirs returns the registered data dirs, deduped and in first-seen
// order. Blank lines and paths that no longer exist on disk are skipped, so a
// deleted data repo silently drops out of the push rotation. A missing registry
// is not an error — it returns an empty slice.
func RegisteredDataDirs() ([]string, error) {
	path, err := RegistryPath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening registry: %w", err)
	}
	defer func() { _ = f.Close() }()

	var dirs []string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		if info, statErr := os.Stat(line); statErr != nil || !info.IsDir() {
			continue
		}
		dirs = append(dirs, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading registry: %w", err)
	}
	return dirs, nil
}
