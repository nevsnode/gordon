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
		t.Log("Basepath() should return an absolute path to a relative file")
		t.Log("returned value:", Basepath(file))
		t.Fail()
	}

	file = "/tmp/foo/../testfile"
	withBase = "/tmp/testfile"
	if withBase != Basepath(file) {
		t.Log("Basepath() should return an absolute path to a relative file")
		t.Log("returned value:", filepath.Clean(Basepath(file)))
		t.Fail()
	}

	file = "/tmp/file"
	if file != Basepath(file) {
		t.Log("Basepath() should return the full file if it is an absolute path")
		t.Log("returned value:", Basepath(file))
		t.Fail()
	}
}
