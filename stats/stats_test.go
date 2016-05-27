package stats

import (
	"encoding/json"
	"github.com/nevsnode/gordon/config"
	"net/http"
	"reflect"
	"testing"
	"time"
)

var testTaskType = "mytask"

func TestStats(t *testing.T) {
	now := getNowUnix()
	tnow := time.Now().Unix()
	if tnow != now {
		t.Log("getNowUnix() should return the current unix timestamp")
		t.FailNow()
	}
	if tnow != runtimeStart {
		t.Log("runtimeStart should have the current unix timestamp")
		t.FailNow()
	}

	sr := getStatsDelayed()
	if len(sr.TaskCount) != 0 {
		t.Log("The initial TaskCount length should be 0")
		t.Fail()
	}

	tasks := make(map[string]config.Task)
	tasks[testTaskType] = config.Task{
		Type: testTaskType,
	}
	InitTasks(tasks)
	sr = getStatsDelayed()
	if len(sr.TaskCount) != 1 {
		t.Log("After initializing one task TaskCount length should be 1")
		t.Fail()
	}
	if sr.TaskCount[testTaskType] != 0 {
		t.Log("The task-count for the initialized task should be 0")
		t.Fail()
	}

	IncrTaskCount(testTaskType)
	sr = getStatsDelayed()
	if sr.TaskCount[testTaskType] != 1 {
		t.Log("The task-count after incrementing should be 1")
		t.Fail()
	}

	IncrTaskCount(testTaskType)
	sr = getStats()
	if sr.TaskCount[testTaskType] < 1 {
		t.Log("The task-count after incrementing should be at least 1")
		t.Fail()
	}

	time.Sleep(1 * time.Second)
	sr = getStats()
	if sr.Runtime < 1 {
		t.Log("statsResponse.Runtime should be greater than 1, after waiting 1 second")
		t.Fail()
	}
}

func getStatsDelayed() statsResponse {
	time.Sleep(50 * time.Millisecond)
	return getStats()
}

func TestStatsHttp(t *testing.T) {
	iface := "127.0.0.1:3333"
	pattern := "/testpattern"
	c := config.StatsConfig{
		Interface: iface,
		Pattern:   pattern,
	}
	go Serve(c)
	time.Sleep(1 * time.Second)

	InitTask(testTaskType)
	IncrTaskCount(testTaskType)

	sr := getStatsDelayed()

	url := "http://" + iface + pattern
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		t.Log("http.Get should not return an error")
		t.Log("err: ", err)
		t.FailNow()
	}

	sr2 := statsResponse{}
	parser := json.NewDecoder(resp.Body)
	err = parser.Decode(&sr2)
	if err != nil {
		t.Log("json.Decode() should not return an error")
		t.Log("err: ", err)
		t.FailNow()
	}
	if !reflect.DeepEqual(sr, sr2) {
		t.Log("getStats() should return the same value as the parsed http-response")
		t.Fail()
	}

	url = "http://" + iface + "/doesnotexist"
	resp, err = http.Get(url)
	if err != nil {
		t.Log("http.Get should not return an error")
		t.Log("err: ", err)
		t.FailNow()
	}
	if resp.StatusCode != 404 {
		t.Log("The HTTP status-code for an invalid path should be 404")
		t.Fail()
	}
}
