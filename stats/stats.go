// Package stats provides functionality for basic usage statistics in Gordon.
package stats

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Stats provides routines to gather basic statistics and serve them through a HTTP-server.
type Stats struct {
	runtimeStart  int64
	taskCount     map[string]int64
	gordonVersion string
}

// statsResponse is the response that will be returned from the HTTP-server,
// containing the statistical data.
type statsResponse struct {
	Runtime   int64            `json:"runtime"`
	TaskCount map[string]int64 `json:"task_count"`
	Version   string           `json:"version"`
}

// NewStats returns a new instance of Stats.
func NewStats() Stats {
	return Stats{
		runtimeStart: getNowUnix(),
		taskCount:    make(map[string]int64),
	}
}

// SetVersion updates the version-number of the Gordon application.
func (s *Stats) SetVersion(version string) {
	s.gordonVersion = version
}

// InitTask initialises the task-counter for a certain task.
// The counter should be initialised so that it will be returned in the HTTP response,
// even when it is 0.
func (s *Stats) InitTask(task string) {
	s.taskCount[task] = 0
}

// IncrTaskCount increments the counter of a certain task.
func (s *Stats) IncrTaskCount(task string) {
	s.taskCount[task]++
}

// getStats returns an instance of the statsResponse which the HTTP-server should reply with.
func (s Stats) getStats() statsResponse {
	return statsResponse{
		Runtime:   s.getRuntime(),
		TaskCount: s.taskCount,
		Version:   s.gordonVersion,
	}
}

// getRuntime returns the runtime of the application in seconds
func (s Stats) getRuntime() int64 {
	return getNowUnix() - s.runtimeStart
}

// getNowUnix returns the current unix timestamp
func getNowUnix() int64 {
	return time.Now().Unix()
}

// ServeHttp spawns an HTTP-server, that responds with a statsResponse in JSON.
func (s Stats) ServeHttp(iface string, pattern string) error {
	http.HandleFunc(pattern, s.httpHandle)
	return http.ListenAndServe(iface, nil)
}

// ServeHttps spawns an HTTP-server, that responds with a statsResponse in JSON, expecting HTTPS connections.
func (s Stats) ServeHttps(iface string, pattern string, cert string, key string) error {
	http.HandleFunc(pattern, s.httpHandle)
	return http.ListenAndServeTLS(iface, cert, key, nil)
}

// httpHandle is the handler that actually responds with the JSON statsResponse.
func (s Stats) httpHandle(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(s.getStats())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf8")
	fmt.Fprintf(w, fmt.Sprintf("%s", b))
}
