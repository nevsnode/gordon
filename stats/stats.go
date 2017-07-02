// Package stats provides functionality for basic usage statistics in Gordon.
package stats

import (
	"encoding/json"
	"fmt"
	"github.com/nevsnode/gordon/config"
	"github.com/nevsnode/gordon/output"
	"github.com/newrelic/go-agent"
	"net/http"
	"sync"
	"time"
)

const httpPath = "/"

var (
	// GordonVersion contains the current version of gordon
	GordonVersion = ""

	runtimeStart = getNowUnix()
	taskCounter  = newTaskCount()
	newRelicApp  newrelic.Application
)

// statsResponse is the response that will be returned from the HTTP-server,
// containing the statistical data.
type statsResponse struct {
	Runtime   int64            `json:"runtime"`
	TaskCount map[string]int64 `json:"task_count"`
	Version   string           `json:"version"`
}

// InitTasks initialises the counters for the defined task-types.
func InitTasks(tasks map[string]config.Task) {
	for taskType := range tasks {
		taskCounter.Init(taskType)
	}
}

// StartedTask handles stats when a task was started.
func StartedTask(task string) Transaction {
	taskCounter.Increment(task)
	return NewTransaction(task)
}

// Setup will initialize the stats-package to be able to record
// statistics within the taskqueue application.
func Setup(c config.StatsConfig) {
	if c.Interface != "" {
		go func() {
			if err := serve(c); err != nil {
				output.NotifyError("stats.serve():", err)
			}
		}()
	}

	if c.NewRelic.License != "" {
		output.Debug("Starting NewRelic Agent")
		nrc := newrelic.NewConfig(c.NewRelic.AppName, c.NewRelic.License)

		var err error
		newRelicApp, err = newrelic.NewApplication(nrc)
		if err != nil {
			output.NotifyError("newrelic.NewApplication():", err)
		}
	}
}

func serve(c config.StatsConfig) error {
	return serveHTTP(c.Interface, httpPath)
}

func serveHTTP(iface string, pattern string) error {
	http.HandleFunc(pattern, httpHandle)
	return http.ListenAndServe(iface, nil)
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
	return statsResponse{
		Runtime:   getRuntime(),
		TaskCount: taskCounter.GetTaskCount(),
		Version:   GordonVersion,
	}
}

func getRuntime() int64 {
	return getNowUnix() - runtimeStart
}

func getNowUnix() int64 {
	return time.Now().Unix()
}

func newTaskCount() *taskCount {
	return &taskCount{
		counts: make(map[string]int64),
	}
}

type taskCount struct {
	counts map[string]int64
	mutex  sync.RWMutex
}

func (t *taskCount) Init(task string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.counts[task] = 0
}

func (t *taskCount) Increment(task string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.counts[task]++
}

func (t *taskCount) GetTaskCount() map[string]int64 {
	t.mutex.RLock()
	t.mutex.RUnlock()
	return t.counts
}

// NewTransaction creates and returns a new transaction instance.
func NewTransaction(name string) (t Transaction) {
	if newRelicApp != nil {
		t.hasNrTxn = true
		t.nrTxn = newRelicApp.StartTransaction(name, nil, nil)
	}

	return
}

// A Transaction allows tracking executions of tasks.
type Transaction struct {
	hasNrTxn bool
	nrTxn    newrelic.Transaction
}

// End will mark the end of the execution of a task.
func (t Transaction) End() {
	if !t.hasNrTxn {
		return
	}

	t.nrTxn.End()
}

// NoticeError will mark the transaction as erroneous and will add the provided error
// to the transaction information.
func (t Transaction) NoticeError(err error) {
	if !t.hasNrTxn {
		return
	}

	t.nrTxn.NoticeError(err)
}
