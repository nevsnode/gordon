package output

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testOutput = ""

func resetTestOutput() {
	testOutput = ""
}

type testLogger struct{}

func (t testLogger) Println(input ...interface{}) {
	testOutput = fmt.Sprint(input...)
}

func TestOutput(t *testing.T) {
	out := New()
	l := testLogger{}
	out.logger = l
	msg := "test output"

	out.SetDebug(true)
	assert.True(t, out.debug, "debug should be true")

	out.SetDebug(false)
	assert.False(t, out.debug, "debug should be false")

	resetTestOutput()
	out.SetDebug(false)
	out.Debug(msg)

	assert.Equal(t, "", testOutput, "output should be empty")

	resetTestOutput()
	out.SetDebug(true)
	out.Debug(msg)

	assert.Equal(t, msg, testOutput, "output should be msg")
}

func TestOutputNotify(t *testing.T) {
	out := New()
	l := testLogger{}
	out.logger = l
	msg := "test output"

	assert.Equal(t, "", out.notifyCmd, "notifyCmd should be empty")

	resetTestOutput()
	out.notify(msg)

	assert.Equal(t, fmt.Sprintf(outputCmdError, emptyCmdError), testOutput, "notify() error print should be emptyCmdError")

	notifyCmd := "echo %s >> /dev/null"
	out.SetNotifyCmd(notifyCmd)

	assert.Equal(t, notifyCmd, out.notifyCmd, "out.notifyCmd should be notifyCmd")

	resetTestOutput()
	out.notify(msg)

	assert.Equal(t, "", testOutput, "notify() should not create output on valid command")

	resetTestOutput()
	notifyCmd = "echo %s"
	out.SetNotifyCmd(notifyCmd)
	out.notify(msg)

	assert.NotEqual(t, "", testOutput, "notify() should create output when command created output")
}