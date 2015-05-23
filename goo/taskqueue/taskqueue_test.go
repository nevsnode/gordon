package taskqueue

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQueueTask(t *testing.T) {
	qt := queueTask{}

	script := "/usr/bin/printf"
	msg := "test output"

	qt.Args = make([]string, 1)
	qt.Args[0] = ""
	err := qt.execute(script)

	assert.Nil(t, err, "err should be nil")

	qt.Args[0] = msg
	err = qt.execute(script)

	assert.Equal(t, msg, fmt.Sprint(err), "error message should be the same")

	jsonString, err := qt.getJsonString()
	jsonStringExpected := "{\"Args\":[\"" + msg + "\"]}"

	assert.Nil(t, err, "err should be nil")
	assert.Equal(t, jsonStringExpected, jsonString, "jsonString should be same as expected")
}

func TestParseQueueTask(t *testing.T) {
	validJson := `{"Args":["_valid"]}`
	invalidJson := "[]"

	_, err := parseQueueTask(invalidJson)

	assert.NotNil(t, err, "err should not be nil")

	pqt, err := parseQueueTask(validJson)

	assert.Nil(t, err, "err should be nil")

	qt := queueTask{}
	qt.Args = make([]string, 1)
	qt.Args[0] = "_valid"

	assert.Equal(t, qt, pqt, "parsed queue-task should be the same")
}
