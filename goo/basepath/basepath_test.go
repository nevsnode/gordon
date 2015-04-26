package basepath

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBasepath(t *testing.T) {
	base, err := New()

	require.Nil(t, err, "error should be nil")

	assert.NotEqual(t, "", base.Path, "Path should not be an empty string")

	assert.Equal(t, base.Path, base.GetPath(), "Path and GetPath() should be the same")

	file := "./testfile"
	withBase := base.Path + "/" + file
	assert.Equal(t, withBase, base.GetPathWith(file), "GetPathWith() should prepend the base path, when file is relative")

	file = "/tmp/file"
	assert.Equal(t, file, base.GetPathWith(file), "GetPathWith() should return the full file if its an absolute path")
}
