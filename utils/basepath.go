package utils

import (
	"os"
	"path/filepath"
)

var root string

func init() {
	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		root = ""
	}
	root = path
}

// Basepath returns an absolute path to the given file.
// If the given file is relative the current absolute path will be prepended.
func Basepath(file string) string {
	if filepath.IsAbs(file) || root == "" {
		return file
	}

	return filepath.Clean(root + "/" + file)
}
