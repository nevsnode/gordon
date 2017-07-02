package taskqueue

import (
	"reflect"
	"testing"
)

func TestQueueTask(t *testing.T) {
	qt := QueueTask{}

	script := "/usr/bin/printf"
	msg := "test output"

	qt.Args = make([]string, 1)
	qt.Args[0] = ""
	err := qt.Execute(script)
	if err != nil {
		t.Log("QueueTask.Execute() should not return an error")
		t.Log("err: ", err)
		t.FailNow()
	}

	qt.Args[0] = msg
	err = qt.Execute(script)
	if msg != err.Error() {
		t.Log("Returned error-message should be the same as the first argument")
		t.Log("err: ", err)
		t.FailNow()
	}

	jsonString, err := qt.GetJSONString()
	jsonStringExpected := `{"args":["` + msg + `"],"env":{},"error_message":""}`
	if err != nil {
		t.Log("QueueTask.GetJSONString() should not return an error")
		t.Log("err: ", err)
		t.Fail()
	}
	if jsonString != jsonStringExpected {
		t.Log("QueueTask.GetJSONString() should return the expected string")
		t.Log("Expected:", jsonStringExpected)
		t.Log("Returned:", jsonString)
		t.Fail()
	}

	qt2 := QueueTask{
		Env: map[string]string{"TEST_ENV_VAR": msg},
	}
	err = qt2.Execute("../testdata/echoenv.sh")
	if msg != err.Error() {
		t.Log("Returned error-message should be the same as the environment variable")
		t.Log("err: ", err)
		t.FailNow()
	}

	jsonString, err = qt2.GetJSONString()
	jsonStringExpected = `{"args":[],"env":{"TEST_ENV_VAR":"` + msg + `"},"error_message":""}`
	if err != nil {
		t.Log("QueueTask.GetJSONString() should not return an error")
		t.Log("err: ", err)
		t.Fail()
	}
	if jsonString != jsonStringExpected {
		t.Log("QueueTask.GetJSONString() should return the expected string")
		t.Log("Expected:", jsonStringExpected)
		t.Log("Returned:", jsonString)
		t.Fail()
	}
}

func TestParseQueueTask(t *testing.T) {
	invalidJSON := "[]"
	_, err := NewQueueTask(invalidJSON)
	if err == nil {
		t.Log("QueueTask.NewQueueTask() should return an error when using invalid JSON")
		t.Fail()
	}

	validJSON := `{}`
	pqt, err := NewQueueTask(validJSON)
	if err != nil {
		t.Log("QueueTask.NewQueueTask() should not return an error when using valid JSON")
		t.Fail()
	}

	qt := QueueTask{}
	if !reflect.DeepEqual(pqt, qt) {
		t.Log("The parsed and the created QueueTask should be the same")
		t.Fail()
	}

	validJSON = `{"args":["_valid"]}`
	pqt, err = NewQueueTask(validJSON)
	if err != nil {
		t.Log("QueueTask.NewQueueTask() should not return an error when using valid JSON")
		t.Fail()
	}

	qt = QueueTask{}
	qt.Args = make([]string, 1)
	qt.Args[0] = "_valid"
	if !reflect.DeepEqual(pqt, qt) {
		t.Log("The parsed and the created QueueTask should be the same")
		t.Fail()
	}

	validJSON = `{"env":{"var1":"val1"}}`
	pqt, err = NewQueueTask(validJSON)
	if err != nil {
		t.Log("QueueTask.NewQueueTask() should not return an error when using valid JSON")
		t.Fail()
	}

	qt = QueueTask{}
	qt.Env = make(map[string]string)
	qt.Env["var1"] = "val1"
	if !reflect.DeepEqual(pqt, qt) {
		t.Log("The parsed and the created QueueTask should be the same")
		t.Fail()
	}

	validJSON = `{"args":["_valid"],"env":{"var1":"val1"}}`
	pqt, err = NewQueueTask(validJSON)
	if err != nil {
		t.Log("QueueTask.NewQueueTask() should not return an error when using valid JSON")
		t.Fail()
	}

	qt = QueueTask{}
	qt.Args = make([]string, 1)
	qt.Args[0] = "_valid"
	qt.Env = make(map[string]string)
	qt.Env["var1"] = "val1"
	if !reflect.DeepEqual(pqt, qt) {
		t.Log("The parsed and the created QueueTask should be the same")
		t.Fail()
	}
}
