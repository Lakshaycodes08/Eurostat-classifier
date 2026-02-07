// fs.go provides directory creation and project-root resolution for the CLI and kernel.
package util

import (
	"os"
	"path/filepath"
)

// EnsureDir ensures that the given directory exists, creating it
// (and any missing parents) with the provided permissions if needed.
func EnsureDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// ProjectRoot returns the current working directory.
// In the future this can be extended to walk up for markers
// (e.g. go.mod, .git, .swytchcode) if needed.
func ProjectRoot() (string, error) {
	return os.Getwd()
}

// Join is a thin wrapper around filepath.Join for convenience.
func Join(elem ...string) string {
	return filepath.Join(elem...)
}

