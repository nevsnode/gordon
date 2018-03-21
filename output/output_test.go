package output

import (
	"fmt"
	"github.com/nevsnode/gordon/config"
	"github.com/nevsnode/gordon/utils"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

var testOutput = ""

func resetTestOutput() {
	testOutput = ""
	errorScript = ""
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

func TestOutputNotifyErrorScript(t *testing.T) {
	l := testLogger{}
	logger = l
	msg := "test output"
	env := make(map[string]string)

	if errorScript != "" {
		t.Log("errorScript should be empty by default")
		t.Fail()
	}

	resetTestOutput()
	notifyErrorScript(errorScript, env, msg)
	if testOutput != "" {
		t.Log("notifyErrorScript() should not create any output when errorScript is empty")
		t.Fail()
	}

	es := "/bin/true"
	SetErrorScript(es)
	if es != errorScript {
		t.Log("errorScript should be the value that was set through SetErrorScript")
		t.FailNow()
	}

	resetTestOutput()
	SetErrorScript(es)
	notifyErrorScript(errorScript, env, msg)
	if testOutput != "" {
		t.Log("notifyErrorScript() should not create output when executing a valid/successful command")
		t.Fail()
	}

	resetTestOutput()
	es = "/bin/cat"
	SetErrorScript(es)
	notifyErrorScript(errorScript, env, msg)
	if testOutput == "" {
		t.Log("notifyErrorScript() should create output when executing a command that created output")
		t.Fail()
	}
	expected := "[ERROR] /bin/cat failed:\n\ttest output\n"
	if testOutput != expected {
		t.Log(fmt.Sprintf("notifyErrorScript() should create '%s' but created '%s' when error-script creates output", expected, testOutput))
		t.Fail()
	}

	resetTestOutput()
	es = "/bin/nonexistent"
	SetErrorScript(es)
	notifyErrorScript(errorScript, env, msg)
	expected = "[ERROR] error_script failed:\n\tfork/exec /bin/nonexistent: no such file or directory\n"
	if testOutput != expected {
		t.Log(fmt.Sprintf("notifyErrorScript() should create '%s' but created '%s' when error-script creates output", expected, testOutput))
		t.Fail()
	}

	resetTestOutput()
	env["TEST_ENV_VAR"] = "foobar"
	notifyErrorScript("../testdata/echoenv.sh", env, msg)
	expected = "[ERROR] ../testdata/echoenv.sh failed:\n\tfoobar\n"
	if testOutput != expected {
		t.Log(fmt.Sprintf("notifyErrorScript() should create '%s' but created '%s' when error-script creates output", expected, testOutput))
		t.Fail()
	}
}

func TestOutputNotifyError(t *testing.T) {
	l := testLogger{}
	logger = l
	msg := "test output"
	var expected string

	resetTestOutput()
	expected = "[ERROR] test output\n"
	NotifyError(msg)
	if testOutput != expected {
		t.Log(fmt.Sprintf("NotifyError() should create '%s' but created '%s'", expected, testOutput))
		t.Fail()
	}
}

func TestOutputNotifyTaskError(t *testing.T) {
	l := testLogger{}
	logger = l
	msg := "test output"
	var expected string
	ct := config.Task{
		Type: "test_type",
	}

	resetTestOutput()
	expected = "[ERROR] test_type\n\t{}\n\ttest output\n"
	NotifyTaskError(ct, "{}", msg)
	if testOutput != expected {
		t.Log(fmt.Sprintf("NotifyError() should create '%s' but created '%s'", expected, testOutput))
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
