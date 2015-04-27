package stats

import (
	"../output"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

var testTaskType = "mytask"

func TestStats(t *testing.T) {
	s := New()
	now := getNowUnix()
	tnow := time.Now().Unix()

	assert.Equal(t, tnow, now, "getNowUnix() should return the same timestamp")

	assert.Equal(t, tnow, s.runtimeStart, "runtimeStart should have the same timestamp")

	sr := s.getStats()

	assert.Len(t, sr.TaskCount, 0, "TaskCount length should be 0")

	s.InitTaskCount(testTaskType)
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
	s := New()
	out := output.New()
	iface := "127.0.0.1:3333"
	go s.ServeHttp(iface, out)

	s.InitTaskCount(testTaskType)
	s.IncrTaskCount(testTaskType)

	sr := s.getStats()

	url := "http://" + iface
	resp, err := http.Get(url)
	defer resp.Body.Close()

	assert.Nil(t, err, "err from http.Get should be nil")

	sr2 := statsResponse{}
	parser := json.NewDecoder(resp.Body)
	err = parser.Decode(&sr2)

	assert.Nil(t, err, "err from parser.Decode() should be nil")

	assert.Equal(t, sr, sr2, "getStats() should return the same as the parsed http-response")
}
