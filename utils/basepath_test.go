package utils

import (
	"path/filepath"
	"testing"
)

func TestBasepath(t *testing.T) {
	if root == "" {
		t.Log("root should not be empty")
		t.FailNow()
	}

	file := "./testfile"
	withBase := filepath.Clean(root + "/" + file)
	if withBase != Basepath(file) {
		t.Log("With() should return an absolute path to a relative file")
		t.Fail()
	}

	file = "/tmp/file"
	if file != Basepath(file) {
		t.Log("With() should return the full file if it is an absolute path")
		t.Fail()
	}
}
