package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ConfigFileName is the per-repo (or global) config file name.
const ConfigFileName = ".jotter"

// Config is the parsed contents of a .jotter file.
type Config struct {
	DataDir string `toml:"data_dir"`
}

// startFromDir returns the directory to begin the walk from. Overridable in
// tests to avoid depending on the real cwd / home dir.
var startFromDir = func() (string, error) {
	return os.Getwd()
}

// homeConfigPath returns the path to ~/.jotter, or empty if the home dir
// cannot be determined. Overridable in tests.
var homeConfigPath = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ConfigFileName)
}

// ResolveConfigFile walks upward from startDir looking for a .jotter file.
// Falls back to ~/.jotter if nothing is found on the walk. Returns the
// absolute path to the chosen file.
func ResolveConfigFile(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, ConfigFileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	home := homeConfigPath()
	if home != "" {
		if info, err := os.Stat(home); err == nil && !info.IsDir() {
			return home, nil
		}
	}
	return "", fmt.Errorf("no %s file found walking up from %s or at %s", ConfigFileName, startDir, home)
}

// LoadConfig reads and parses a .jotter file. The data_dir value is expanded:
// a leading ~ becomes the user's home dir, and relative paths are resolved
// against the directory containing the config file.
func LoadConfig(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.DataDir == "" {
		return nil, fmt.Errorf("%s: data_dir is required", path)
	}
	cfg.DataDir = expandPath(cfg.DataDir, filepath.Dir(path))
	return &cfg, nil
}

// GetDataDir resolves the session logs data directory for the current cwd.
func GetDataDir() (string, error) {
	startDir, err := startFromDir()
	if err != nil {
		return "", err
	}
	path, err := ResolveConfigFile(startDir)
	if err != nil {
		return "", err
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		return "", err
	}
	return cfg.DataDir, nil
}

// ResolveConfig returns both the config file path and the parsed config for
// the current cwd. Useful for `jotter config` which wants to display both.
func ResolveConfig() (string, *Config, error) {
	startDir, err := startFromDir()
	if err != nil {
		return "", nil, err
	}
	path, err := ResolveConfigFile(startDir)
	if err != nil {
		return "", nil, err
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		return path, nil, err
	}
	return path, cfg, nil
}

// expandPath resolves ~ to the user's home dir and resolves relative paths
// against baseDir.
func expandPath(p, baseDir string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(baseDir, p)
	}
	return p
}
