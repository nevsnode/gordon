// Package basepath provides functionality to get the absolute path to a certain file.
package basepath

import (
	"os"
	"path/filepath"
)

// GetPathWith returns an absolute path to the given file.
// If the given file is relative the current absolute path will be prepended.
func With(file string) string {
	if filepath.IsAbs(file) {
		return file
	}

	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return file
	}

	return filepath.Clean(path + "/" + file)
}
