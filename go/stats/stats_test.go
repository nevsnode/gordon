package stats

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

var testTaskType = "mytask"

func TestStats(t *testing.T) {
	s := NewStats()
	now := getNowUnix()
	tnow := time.Now().Unix()

	assert.Equal(t, tnow, now, "getNowUnix() should return the same timestamp")

	assert.Equal(t, tnow, s.runtimeStart, "runtimeStart should have the same timestamp")

	sr := s.getStats()

	assert.Len(t, sr.TaskCount, 0, "TaskCount length should be 0")

	s.InitTask(testTaskType)
	sr = s.getStats()

	assert.Len(t, sr.TaskCount, 1, "TaskCount length after first init should be 1")
	assert.EqualValues(t, 0, sr.TaskCount[testTaskType], "initialized taskcount should be 0")

	s.IncrTaskCount(testTaskType)
	sr = s.getStats()

	assert.EqualValues(t, 1, sr.TaskCount[testTaskType], "incremented taskcount should be 1")

	time.Sleep(1 * time.Second)
	sr = s.getStats()

	assert.True(t, sr.Runtime >= 1, "Runtime should be greater than one second")
}

func TestStatsHttp(t *testing.T) {
	s := NewStats()
	iface := "127.0.0.1:3333"
	pattern := "/testpattern"
	go s.ServeHttp(iface, pattern)

	s.InitTask(testTaskType)
	s.IncrTaskCount(testTaskType)

	sr := s.getStats()

	url := "http://" + iface + pattern
	resp, err := http.Get(url)
	defer resp.Body.Close()

	assert.Nil(t, err, "err from http.Get should be nil")

	sr2 := statsResponse{}
	parser := json.NewDecoder(resp.Body)
	err = parser.Decode(&sr2)

	assert.Nil(t, err, "err from parser.Decode() should be nil")

	assert.Equal(t, sr, sr2, "getStats() should return the same as the parsed http-response")

	url = "http://" + iface + "/doesnotexist"
	resp, err = http.Get(url)

	assert.Nil(t, err, "err from http.Get should be nil")

	assert.Equal(t, 404, resp.StatusCode, "StatusCode from response should be 404 (Not found)")
}