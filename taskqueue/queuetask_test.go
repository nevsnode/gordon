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
	jsonStringExpected := `{"args":["` + msg + `"],"error_message":""}`
	if err != nil {
		t.Log("QueueTask.GetJSONString() should not return an error")
		t.Log("err: ", err)
		t.Fail()
	}
	if jsonString != jsonStringExpected {
		t.Log("QueueTask.GetJSONString() should return the expected string")
		t.Log("Expected: ", jsonStringExpected)
		t.Fail()
	}
}

func TestParseQueueTask(t *testing.T) {
	validJSON := `{"args":["_valid"]}`
	invalidJSON := "[]"

	_, err := NewQueueTask(invalidJSON)
	if err == nil {
		t.Log("QueueTask.NewQueueTask() should return an error when using invalid JSON")
		t.Fail()
	}

	pqt, err := NewQueueTask(validJSON)
	if err != nil {
		t.Log("QueueTask.NewQueueTask() should not return an error when using valid JSON")
		t.Fail()
	}

	qt := QueueTask{}
	qt.Args = make([]string, 1)
	qt.Args[0] = "_valid"
	if !reflect.DeepEqual(pqt, qt) {
		t.Log("The parsed and the created QueueTask should be the same")
		t.Fail()
	}
}
