package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDataDir_FromEnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JOTTER_DATA", dir)

	got, err := GetDataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != dir {
		t.Errorf("got %q, want %q", got, dir)
	}
}

func TestGetDataDir_MissingEnvAndNoConfig_ReturnsError(t *testing.T) {
	t.Setenv("JOTTER_DATA", "")
	// Override config path to a non-existent location
	origConfigPath := configFilePath
	configFilePath = filepath.Join(t.TempDir(), "nonexistent", "config")
	defer func() { configFilePath = origConfigPath }()

	_, err := GetDataDir()
	if err == nil {
		t.Fatal("expected error when JOTTER_DATA is unset and no config file exists")
	}
}

func TestGetDataDir_FallsBackToConfigFile(t *testing.T) {
	t.Setenv("JOTTER_DATA", "")

	dir := t.TempDir()
	configDir := filepath.Join(t.TempDir(), ".config", "jotter")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(configDir, "config")
	if err := os.WriteFile(configFile, []byte(dir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origConfigPath := configFilePath
	configFilePath = configFile
	defer func() { configFilePath = origConfigPath }()

	got, err := GetDataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != dir {
		t.Errorf("got %q, want %q", got, dir)
	}
}
