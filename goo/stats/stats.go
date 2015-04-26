package stats

import (
	"../output"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Stats struct {
	runtimeStart int64
	taskCount    map[string]int64
}

type StatsResponse struct {
	Runtime   int64
	TaskCount map[string]int64
}

func New() Stats {
	return Stats{
		runtimeStart: getNowUnix(),
		taskCount:    make(map[string]int64),
	}
}

func (s *Stats) InitTaskCount(task string) {
	s.taskCount[task] = 0
}

func (s *Stats) IncrTaskCount(task string) {
	s.taskCount[task]++
}

func (s Stats) getStats() StatsResponse {
	return StatsResponse{
		Runtime:   s.getRuntime(),
		TaskCount: s.taskCount,
	}
}

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

func (s Stats) getRuntime() int64 {
	return getNowUnix() - s.runtimeStart
}

func getNowUnix() int64 {
	return time.Now().Unix()
}
