package util

import (
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
)

// createToolingJSON creates .swytchcode/tooling.json inside dir.
func createToolingJSON(t *testing.T, dir string) {
	t.Helper()
	swytchDir := filepath.Join(dir, constants.SwytchDirName)
	if err := os.MkdirAll(swytchDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	toolingPath := filepath.Join(swytchDir, constants.ToolingJSONFile)
	if err := os.WriteFile(toolingPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// chdirTemp changes the working directory to dir and restores the original on cleanup.
func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

func TestProjectRoot_FoundAtCwd(t *testing.T) {
	root := t.TempDir()
	createToolingJSON(t, root)
	chdirTemp(t, root)

	got, err := ProjectRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Resolve symlinks so TempDir paths compare equal on macOS (/var → /private/var).
	want, _ := filepath.EvalSymlinks(root)
	got, _ = filepath.EvalSymlinks(got)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProjectRoot_FoundInParent(t *testing.T) {
	root := t.TempDir()
	createToolingJSON(t, root)
	sub := filepath.Join(root, "server")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	chdirTemp(t, sub)

	got, err := ProjectRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := filepath.EvalSymlinks(root)
	got, _ = filepath.EvalSymlinks(got)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProjectRoot_FoundInGrandparent(t *testing.T) {
	root := t.TempDir()
	createToolingJSON(t, root)
	deep := filepath.Join(root, "server", "src")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	chdirTemp(t, deep)

	got, err := ProjectRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := filepath.EvalSymlinks(root)
	got, _ = filepath.EvalSymlinks(got)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProjectRoot_NotFound_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	chdirTemp(t, sub)

	_, err := ProjectRoot()
	if err == nil {
		t.Fatal("expected error when no .swytchcode/tooling.json in tree")
	}
}

func TestInitProjectRoot_NotFound_UsesCwd(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	chdirTemp(t, sub)

	got, err := InitProjectRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := filepath.EvalSymlinks(sub)
	got, _ = filepath.EvalSymlinks(got)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
