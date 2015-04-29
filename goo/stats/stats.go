// Package stats provides functionality for basic usage statistics in Goophry.
package stats

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Stats provides routines to gather basic statistics and serve them through a HTTP-server.
type Stats struct {
	runtimeStart int64
	taskCount    map[string]int64
}

// statsResponse is the response that will be returned from the HTTP-server,
// containing the statistical data.
type statsResponse struct {
	Runtime   int64
	TaskCount map[string]int64
}

// NewStats returns a new instance of Stats.
func NewStats() Stats {
	return Stats{
		runtimeStart: getNowUnix(),
		taskCount:    make(map[string]int64),
	}
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
	}
}

// ServeHttp spawns a HTTP-server that responds with a statsResponse in JSON.
// In case of an error, it will use the notify-functionality from output, since this routine
// will likely be run as a go-routine.
func (s Stats) ServeHttp(iface string) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b, err := json.Marshal(s.getStats())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf8")
		fmt.Fprintf(w, fmt.Sprintf("%s", b))
	})

	return http.ListenAndServe(iface, nil)
}

// getRuntime returns the runtime of the application in seconds
func (s Stats) getRuntime() int64 {
	return getNowUnix() - s.runtimeStart
}

// getNowUnix returns the current unix timestamp
func getNowUnix() int64 {
	return time.Now().Unix()
}
