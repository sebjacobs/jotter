package internal

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// useStateDir points StateDir at an isolated temp dir for the test's duration.
func useStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(StateDirEnv, dir)
	return dir
}

// mkDataDir creates a real directory under the test root so registry reads
// (which skip non-existent paths) keep it.
func mkDataDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRegisteredDataDirs_MissingFileIsEmpty(t *testing.T) {
	useStateDir(t)
	dirs, err := RegisteredDataDirs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dirs) != 0 {
		t.Errorf("expected empty, got %v", dirs)
	}
}

func TestRegisterDataDir_AppendsAndDedupes(t *testing.T) {
	useStateDir(t)
	a := mkDataDir(t, "repo-a")
	b := mkDataDir(t, "repo-b")

	for _, d := range []string{a, b, a, b} {
		if err := RegisterDataDir(d); err != nil {
			t.Fatalf("RegisterDataDir(%s): %v", d, err)
		}
	}

	dirs, err := RegisteredDataDirs()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{a, b}; !reflect.DeepEqual(dirs, want) {
		t.Errorf("got %v, want %v", dirs, want)
	}
}

func TestRegisterDataDir_StoresAbsolutePath(t *testing.T) {
	useStateDir(t)
	abs := mkDataDir(t, "repo")
	rel, err := filepath.Rel(mustCwd(t), abs)
	if err != nil {
		t.Skipf("cannot relativise %s: %v", abs, err)
	}

	if err := RegisterDataDir(rel); err != nil {
		t.Fatal(err)
	}
	dirs, err := RegisteredDataDirs()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{abs}; !reflect.DeepEqual(dirs, want) {
		t.Errorf("got %v, want %v", dirs, want)
	}
}

func TestRegisteredDataDirs_SkipsStalePaths(t *testing.T) {
	useStateDir(t)
	live := mkDataDir(t, "live")
	gone := mkDataDir(t, "gone")

	if err := RegisterDataDir(live); err != nil {
		t.Fatal(err)
	}
	if err := RegisterDataDir(gone); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	dirs, err := RegisteredDataDirs()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{live}; !reflect.DeepEqual(dirs, want) {
		t.Errorf("got %v, want %v", dirs, want)
	}
}

func mustCwd(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return cwd
}
