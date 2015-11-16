// Package basepath provides functionality to get the absolute path to a certain file.
package basepath

import (
	"os"
	"path/filepath"
)

var base string

func init() {
	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		base = ""
	}
	base = path
}

// With returns an absolute path to the given file.
// If the given file is relative the current absolute path will be prepended.
func With(file string) string {
	if filepath.IsAbs(file) || base == "" {
		return file
	}

	return filepath.Clean(base + "/" + file)
}
