package output

import (
	"fmt"
	"github.com/nevsnode/gordon/utils"
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
	testOutput = fmt.Sprintln(input...)
}

func TestOutput(t *testing.T) {
	l := testLogger{}
	logger = l
	msg := "test output"

	SetDebug(true)
	if debug != true {
		t.Log("debug should be true")
		t.FailNow()
	}

	SetDebug(false)
	if debug != false {
		t.Log("debug should be false")
		t.FailNow()
	}

	resetTestOutput()
	SetDebug(false)
	Debug(msg)
	if testOutput != "" {
		t.Log("No output should be produced when debug is false")
		t.Fail()
	}

	resetTestOutput()
	SetDebug(true)
	Debug(msg)
	if testOutput != prependDebug+" "+msg+"\n" {
		t.Log("The debug-message should be printed when debug is true")
		t.Fail()
	}

	resetTestOutput()
	Debug("a", "b", 3)
	if testOutput != prependDebug+" a b 3\n" {
		t.Log("Debug() should accept multiple parameters of several types")
		t.Fail()
	}

	resetTestOutput()
	Debug("foo\nbar")
	if testOutput != prependDebug+" foo\n\tbar\n" {
		t.Log("Text over multiple lines should be indented with a tabulator")
		t.Log("Output:", testOutput)
		t.Fail()
	}
}

func TestOutputNotify(t *testing.T) {
	l := testLogger{}
	logger = l
	msg := "test output"

	if errorScript != "" {
		t.Log("errorScript should be empty by default")
		t.Fail()
	}

	resetTestOutput()
	notify(msg)
	if testOutput != "" {
		t.Log("notify() should not create any output when errorScript is empty")
		t.Fail()
	}

	errorScript := "/bin/true"
	SetErrorScript(errorScript)
	if errorScript != errorScript {
		t.Log("errorScript should be the value that was set through SetErrorScript")
		t.FailNow()
	}

	resetTestOutput()
	notify(msg)
	if testOutput != "" {
		t.Log("notify() should not create output when executing a valid/successful command")
		t.Fail()
	}

	resetTestOutput()
	errorScript = "/bin/cat"
	SetErrorScript(errorScript)
	notify(msg)
	if testOutput == "" {
		t.Log("notify() should create output when executing a command that created output")
		t.Fail()
	}
}

func TestOutputLogfile(t *testing.T) {
	path := utils.Basepath("./output.test.log")
	msg := "test output"

	err := SetLogfile(path)
	if err != nil {
		t.Log("SetLogfile() should not return an error")
		t.Log("err: ", err)
		t.FailNow()
	}

	SetDebug(true)
	Debug(msg)

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
