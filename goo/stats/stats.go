// Package stats provides functionality for basic usage statistics in Goophry.
package stats

import (
	"../output"
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

// StatsResponse is the response that will be returned from the HTTP-server,
// containing the statistical data.
type StatsResponse struct {
	Runtime   int64
	TaskCount map[string]int64
}

// New returns a new instance of Stats.
func New() Stats {
	return Stats{
		runtimeStart: getNowUnix(),
		taskCount:    make(map[string]int64),
	}
}

// InitTaskCount initialises the task-counter for a certain task.
// The counter should be initialised so that it will be returned in the HTTP response,
// even when it is 0.
func (s *Stats) InitTaskCount(task string) {
	s.taskCount[task] = 0
}

// IncrTaskCount increments the counter of a certain task.
func (s *Stats) IncrTaskCount(task string) {
	s.taskCount[task]++
}

// getStats returns an instance of the StatsResponse which the HTTP-server should reply with.
func (s Stats) getStats() StatsResponse {
	return StatsResponse{
		Runtime:   s.getRuntime(),
		TaskCount: s.taskCount,
	}
}

// ServeHttp spawns a HTTP-server that responds with a StatsResponse in JSON.
// In case of an error, it will use the notify-functionality from output, since this routine
// will likely be run as a go-routine.
func (s Stats) ServeHttp(iface string, out output.Output) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b, err := json.Marshal(s.getStats())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf8")
		fmt.Fprintf(w, fmt.Sprintf("%s", b))
	})

	err := http.ListenAndServe(iface, nil)
	if err != nil {
		msg := fmt.Sprintf("stats.ServeHttp(): %s", err)
		out.NotifyError(msg)
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
