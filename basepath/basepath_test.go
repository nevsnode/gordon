package basepath

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestBasepath(t *testing.T) {
	base, err := filepath.Abs(filepath.Dir(os.Args[0]))
	require.Nil(t, err, "error should be nil")

	file := "./testfile"
	withBase := filepath.Clean(base + "/" + file)
	assert.Equal(t, withBase, With(file), "GetPathWith() should return an absolute path to the relative file")

	file = "/tmp/file"
	assert.Equal(t, file, With(file), "GetPathWith() should return the full file if its an absolute path")
}
