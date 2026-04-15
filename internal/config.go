package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// configFilePath is the fallback config file location.
// Exported as a var so tests can override it.
var configFilePath = filepath.Join(homeDir(), ".config", "jotter", "config")

// GetDataDir resolves the session logs data directory.
// Checks JOTTER_DATA env var first, then falls back to the config file.
func GetDataDir() (string, error) {
	if dir := os.Getenv("JOTTER_DATA"); dir != "" {
		return dir, nil
	}

	data, err := os.ReadFile(configFilePath)
	if err == nil {
		dir := strings.TrimSpace(string(data))
		if dir != "" {
			return dir, nil
		}
	}

	return "", fmt.Errorf("JOTTER_DATA is not set and no config found at %s", configFilePath)
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
