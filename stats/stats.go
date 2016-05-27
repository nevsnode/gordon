// Package stats provides functionality for basic usage statistics in Gordon.
package stats

import (
	"encoding/json"
	"fmt"
	"github.com/nevsnode/gordon/config"
	"net/http"
	"time"
)

const (
	incrBuffer = 10000
)

var (
	// GordonVersion contains the current version of gordon
	GordonVersion = ""

	runtimeStart = getNowUnix()
	taskCount    = make(map[string]int64)
	incrChan     = make(chan string, incrBuffer)
	getStatsChan = make(chan chan statsResponse)
)

// statsResponse is the response that will be returned from the HTTP-server,
// containing the statistical data.
type statsResponse struct {
	Runtime   int64            `json:"runtime"`
	TaskCount map[string]int64 `json:"task_count"`
	Version   string           `json:"version"`
}

func init() {
	go updateCount()
}

func updateCount() {
	for {
		select {
		case taskType := <-incrChan:
			taskCount[taskType]++
		case response := <-getStatsChan:
			response <- statsResponse{
				Runtime:   getRuntime(),
				TaskCount: taskCount,
				Version:   GordonVersion,
			}
		}
	}
}

// InitTask initialises the task-counter for the defined task-type.
// The counter should be initialised so that it will be returned in the HTTP response,
// even when it is 0.
func InitTask(task string) {
	taskCount[task] = 0
}

// InitTasks initialises the counters for the defined task-types.
func InitTasks(tasks map[string]config.Task) {
	for taskType := range tasks {
		InitTask(taskType)
	}
}

// IncrTaskCount increments the counter of the defined task-type.
func IncrTaskCount(task string) {
	incrChan <- task
}

// Serve handles spawning an appropriate HTTP/HTTPS-server
func Serve(c config.StatsConfig) error {
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		return serveHTTPS(c.Interface, c.Pattern, c.TLSCertFile, c.TLSKeyFile)
	}
	return serveHTTP(c.Interface, c.Pattern)
}

func serveHTTP(iface string, pattern string) error {
	http.HandleFunc(pattern, httpHandle)
	return http.ListenAndServe(iface, nil)
}

func serveHTTPS(iface string, pattern string, cert string, key string) error {
	http.HandleFunc(pattern, httpHandle)
	return http.ListenAndServeTLS(iface, cert, key, nil)
}

func httpHandle(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(getStats())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf8")
	fmt.Fprintf(w, fmt.Sprintf("%s", b))
}

func getStats() statsResponse {
	request := make(chan statsResponse)
	getStatsChan <- request
	return <-request
}

func getRuntime() int64 {
	return getNowUnix() - runtimeStart
}

func getNowUnix() int64 {
	return time.Now().Unix()
}
