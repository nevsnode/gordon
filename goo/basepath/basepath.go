package basepath

import (
	"os"
	"path/filepath"
)

type Basepath struct {
	Path string
}

func New() (b Basepath, err error) {
	b.Path, err = getBasePath()
	return
}

func getBasePath() (string, error) {
	return filepath.Abs(filepath.Dir(os.Args[0]))
}

func (b Basepath) GetPath() string {
	return b.Path
}

func (b Basepath) GetPathWith(file string) string {
	if !filepath.IsAbs(file) {
		file = b.Path + "/" + file
	}
	return file
}
