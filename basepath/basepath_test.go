package basepath

import (
	"path/filepath"
	"testing"
)

func TestBasepath(t *testing.T) {
	if base == "" {
		t.Log("base should not be empty")
		t.FailNow()
	}

	file := "./testfile"
	withBase := filepath.Clean(base + "/" + file)
	if withBase != With(file) {
		t.Log("With() should return an absolute path to a relative file")
		t.Fail()
	}

	file = "/tmp/file"
	if file != With(file) {
		t.Log("With() should return the full file if it is an absolute path")
		t.Fail()
	}
}
