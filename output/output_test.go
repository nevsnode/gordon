package output

import (
	"../basepath"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
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
	if out.debug != true {
		t.Log("output.debug should be true")
		t.FailNow()
	}

	out.SetDebug(false)
	if out.debug != false {
		t.Log("output.debug should be false")
		t.FailNow()
	}

	resetTestOutput()
	out.SetDebug(false)
	out.Debug(msg)
	if testOutput != "" {
		t.Log("No output should be produced when output.debug is false")
		t.Fail()
	}

	resetTestOutput()
	out.SetDebug(true)
	out.Debug(msg)
	if testOutput != msg {
		t.Log("The debug-message should be printed when output.debug is true")
		t.Fail()
	}
}

func TestOutputNotify(t *testing.T) {
	out := New()
	l := testLogger{}
	out.logger = l
	msg := "test output"

	if out.errorScript != "" {
		t.Log("output.errorScript should be empty by default")
		t.Fail()
	}

	resetTestOutput()
	out.notify(msg)
	if testOutput != "" {
		t.Log("output.notify() should not create any output when output.errorScript is empty")
		t.Fail()
	}

	errorScript := "/bin/true"
	out.SetErrorScript(errorScript)
	if errorScript != out.errorScript {
		t.Log("output.errorScript should be the value that was set through output.SetErrorScript")
		t.FailNow()
	}

	resetTestOutput()
	out.notify(msg)
	if testOutput != "" {
		t.Log("output.notify() should not create output when executing a valid/successful command")
		t.Fail()
	}

	resetTestOutput()
	errorScript = "/bin/cat"
	out.SetErrorScript(errorScript)
	out.notify(msg)
	if testOutput == "" {
		t.Log("output.notify() should create output when executing a command that created output")
		t.Fail()
	}
}

func TestOutputLogfile(t *testing.T) {
	path := basepath.With("./output.test.log")
	msg := "test output"

	out := New()
	err := out.SetLogfile(path)
	if err != nil {
		t.Log("output.SetLogfile() should not return an error")
		t.Log("err: ", err)
		t.FailNow()
	}

	out.SetDebug(true)
	out.Debug(msg)

	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.Log("ioutil.ReadFile() should not return an error")
		t.Log("err: ", err)
		t.FailNow()
	}

	if !strings.Contains(string(b), msg) {
		t.Log("logfile should contain the debug-message")
		t.Fail()
	}

	err = os.Remove(path)
	if err != nil {
		t.Log("os.Remove() should not return an error")
		t.Log("err: ", err)
		t.Fail()
	}
}
