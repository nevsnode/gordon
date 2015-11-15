package output

import (
	"../basepath"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
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

	assert.Equal(t, "", out.errorScript, "errorScript should be empty")

	resetTestOutput()
	out.notify(msg)

	assert.Equal(t, "", testOutput, "notify() error print should be empty")

	errorScript := "/bin/true"
	out.SetErrorScript(errorScript)

	assert.Equal(t, errorScript, out.errorScript, "out.errorScript should be errorScript")

	resetTestOutput()
	out.notify(msg)

	assert.Equal(t, "", testOutput, "notify() should not create output on valid command")

	resetTestOutput()
	errorScript = "/bin/cat"
	out.SetErrorScript(errorScript)
	out.notify(msg)

	assert.NotEqual(t, "", testOutput, "notify() should create output when command created output")
}

func TestOutputLogfile(t *testing.T) {
	path := basepath.With("./output.test.log")
	msg := "test output"

	out := New()
	err := out.SetLogfile(path)

	assert.Nil(t, err, "output.SetLogfile() err should be nil")

	out.SetDebug(true)
	out.Debug(msg)

	b, err := ioutil.ReadFile(path)

	assert.Nil(t, err, "ioutil.ReadFile() err should be nil")

	assert.Contains(t, string(b), msg, "logfile should contain the debug message")

	err = os.Remove(path)
	assert.Nil(t, err, "os.Remove() err should be nil")
}
