// Package basepath provides functionality to get the current absolute directory of the application.
// It also provides a method to return an absolute path from a possible relative path.
package basepath

import (
	"os"
	"path/filepath"
)

// A Basepath holds the current absolute directory of the application, and routines to return it.
type Basepath struct {
	Path string
}

// NewBasepath returns a new Basepath instance.
// It may also return an error, when the path could not be determined.
func NewBasepath() (b Basepath, err error) {
	b.Path, err = getBasePath()
	return
}

// getBasePath returns the current absolute directory of the running application
// or an error if something went wrong.
func getBasePath() (string, error) {
	return filepath.Abs(filepath.Dir(os.Args[0]))
}

// GetPath returns the current absolute directory.
func (b Basepath) GetPath() string {
	return b.Path
}

// GetPathWith returns an absolute path to the given file.
// If the given file is relative the current absolute path will be prepended.
func (b Basepath) GetPathWith(file string) string {
	if !filepath.IsAbs(file) {
		file = filepath.Clean(b.Path + "/" + file)
	}
	return file
}
