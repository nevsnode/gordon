package stats

import (
	"../config"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"
)

var testTaskType = "mytask"

func TestStats(t *testing.T) {
	s := New()
	now := getNowUnix()
	tnow := time.Now().Unix()
	if tnow != now {
		t.Log("getNowUnix() should return the current unix timestamp")
		t.FailNow()
	}
	if tnow != s.runtimeStart {
		t.Log("stats.runtimeStart should have the current unix timestamp")
		t.FailNow()
	}

	sr := s.getStats()
	if len(sr.TaskCount) != 0 {
		t.Log("The initial stats.TaskCount length should be 0")
		t.Fail()
	}

	s.InitTask(testTaskType)
	sr = s.getStats()
	if len(sr.TaskCount) != 1 {
		t.Log("After initializing one task stats.TaskCount length should be 1")
		t.Fail()
	}
	if sr.TaskCount[testTaskType] != 0 {
		t.Log("The task-count for the initialized task should be 0")
		t.Fail()
	}

	s.IncrTaskCount(testTaskType)
	sr = s.getStats()
	if sr.TaskCount[testTaskType] != 1 {
		t.Log("The task-count after incrementing should be 1")
		t.Fail()
	}

	time.Sleep(1 * time.Second)
	sr = s.getStats()
	if sr.Runtime < 1 {
		t.Log("statsResponse.Runtime should be greater than 1, after waiting 1 second")
		t.Fail()
	}
}

func TestStatsHttp(t *testing.T) {
	s := New()
	iface := "127.0.0.1:3333"
	pattern := "/testpattern"
	c := config.StatsConfig{Interface: iface, Pattern: pattern}
	go s.Serve(c)
	time.Sleep(1 * time.Second)

	s.InitTask(testTaskType)
	s.IncrTaskCount(testTaskType)

	sr := s.getStats()

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
		t.Log("stats.getStats() should return the same value as the parsed http-response")
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
