package internal

import (
	"os"
	"path/filepath"
	"testing"
)

// withConfigOverrides points startFromDir / homeConfigPath at t.TempDir-backed
// values for the duration of a test. Restores originals on t.Cleanup.
func withConfigOverrides(t *testing.T, startDir, homeConfig string) {
	t.Helper()
	origStart, origHome := startFromDir, homeConfigPath
	startFromDir = func() (string, error) { return startDir, nil }
	homeConfigPath = func() string { return homeConfig }
	t.Cleanup(func() {
		startFromDir = origStart
		homeConfigPath = origHome
	})
}

func writeConfig(t *testing.T, dir, dataDir string) string {
	t.Helper()
	path := filepath.Join(dir, ConfigFileName)
	body := "data_dir = " + quote(dataDir) + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGetDataDir_FindsRepoLevelConfig(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	nested := filepath.Join(repo, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	writeConfig(t, repo, dataDir)

	withConfigOverrides(t, nested, filepath.Join(root, "nope"))

	got, err := GetDataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != dataDir {
		t.Errorf("got %q, want %q", got, dataDir)
	}
}

func TestGetDataDir_FallsBackToHomeConfig(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	// no .jotter in repo or ancestors — fallback should fire

	homeRoot := t.TempDir()
	dataDir := t.TempDir()
	homeConfig := writeConfig(t, homeRoot, dataDir)

	withConfigOverrides(t, repo, homeConfig)

	got, err := GetDataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != dataDir {
		t.Errorf("got %q, want %q", got, dataDir)
	}
}

func TestGetDataDir_RepoConfigWinsOverHome(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	repoData := t.TempDir()
	writeConfig(t, repo, repoData)

	homeRoot := t.TempDir()
	homeData := t.TempDir()
	homeConfig := writeConfig(t, homeRoot, homeData)

	withConfigOverrides(t, repo, homeConfig)

	got, err := GetDataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != repoData {
		t.Errorf("got %q, want %q (repo config should win)", got, repoData)
	}
}

func TestGetDataDir_NoConfigAnywhere_ReturnsError(t *testing.T) {
	withConfigOverrides(t, t.TempDir(), filepath.Join(t.TempDir(), "missing"))

	_, err := GetDataDir()
	if err == nil {
		t.Fatal("expected error when no .jotter file exists anywhere")
	}
}

func TestLoadConfig_RejectsMissingDataDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte("# empty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("expected error for config missing data_dir")
	}
}

func TestLoadConfig_ExpandsTilde(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte(`data_dir = "~/some-logs"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "some-logs")
	if cfg.DataDir != want {
		t.Errorf("got %q, want %q", cfg.DataDir, want)
	}
}

func TestLoadConfig_ResolvesRelativePathAgainstConfigDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte(`data_dir = "./logs"`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "logs")
	if cfg.DataDir != want {
		t.Errorf("got %q, want %q", cfg.DataDir, want)
	}
}
