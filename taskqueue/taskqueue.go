// Package taskqueue provides the functionality for receiving, handling and executing tasks.
// This package provides the routines for the task- and queue-workers.
// Queue-workers are the go-routines that wait for entries in the Redis-lists,
// parse them and send them to the task-workers.
// Task-workers are the go-routines that finally execute the tasks that they receive
// from the queue-workers.
package taskqueue

import (
	"../config"
	"../output"
	"../stats"
	"encoding/json"
	"fmt"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

// A queueTask is the task as it is enqueued in a Redis-list.
type queueTask struct {
	Args         []string `json:"args"`          // list of arguments passed to the defined script/application
	ErrorMessage string   `json:"error_message"` // error message that might be created on executing the task
}

// execute executes the passed script/application with the arguments from the queueTask object.
func (q queueTask) execute(script string) error {
	cmd := exec.Command(script, q.Args...)

	// set Setpgid to true, to execute command in different process group,
	// so it won't receive the interrupt-signals sent to the main go-application
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	out, err := cmd.Output()

	if len(out) != 0 && err == nil {
		err = fmt.Errorf("%s", out)
	}

	return err
}

// getJsonString returns the queueTask object as a json-encoded string
func (q queueTask) getJsonString() (value string, err error) {
	b, err := json.Marshal(q)
	value = fmt.Sprintf("%s", b)
	return
}

// parseQueueTask parses the string-value from a Redis-list entry.
// It returns an object of queueTask and a possible error if parsing failed.
func parseQueueTask(value string) (task queueTask, err error) {
	reader := strings.NewReader(value)
	parser := json.NewDecoder(reader)
	err = parser.Decode(&task)
	return
}

// A Taskqueue offers routines for the task-workers and queue-workers.
// It also offers routines to set the config-, output- and stats-objects,
// which are used from the worker-routines.
type Taskqueue struct {
	waitGroup      sync.WaitGroup // wait group used to handle a proper application shutdown
	config         config.Config  // config object, storing, for instance, connection data
	output         output.Output  // output object for handling debug-/error-messages and notifying about task execution errors
	stats          *stats.Stats   // stats object for gathering usage data
	quit           chan int       // channel used to gracefully shutdown all go-routines
	failedConnPool *pool.Pool     // pool of connections used for inserting failed task into their lists
}

// New returns a new instance of a Taskqueue
func New() Taskqueue {
	return Taskqueue{
		quit: make(chan int),
	}
}

// SetConfig sets the config object
func (tq *Taskqueue) SetConfig(c config.Config) {
	tq.config = c

	var err error
	if tq.config.FailedTasksTTL > 0 {
		tq.failedConnPool, err = pool.New(tq.config.RedisNetwork, tq.config.RedisAddress, len(tq.config.Tasks))
		if err != nil {
			tq.output.StopError(fmt.Sprintf("pool.New(): %s", err))
		}
	}
}

// SetOutput sets the output object
func (tq *Taskqueue) SetOutput(o output.Output) {
	tq.output = o
}

// SetStats sets the stats object
func (tq *Taskqueue) SetStats(s *stats.Stats) {
	tq.stats = s
}

// CreateWorkers creates all worker go-routines.
func (tq *Taskqueue) CreateWorkers(ct config.Task) {
	queue := make(chan queueTask)

	if ct.Workers <= 1 {
		ct.Workers = 1
	}

	for i := 0; i < ct.Workers; i++ {
		tq.waitGroup.Add(1)
		go tq.taskWorker(ct, queue)
	}
	tq.output.Debug(fmt.Sprintf("Created %d workers for type %s", ct.Workers, ct.Type))

	tq.waitGroup.Add(1)
	go tq.queueWorker(ct, queue)
	tq.output.Debug(fmt.Sprintf("Created queue worker for type %s", ct.Type))
}

// Wait waits for the waitGroup to keep the application running, for as long as there
// are any go-routines active.
func (tq *Taskqueue) Wait() {
	tq.waitGroup.Wait()

	if tq.failedConnPool != nil {
		tq.failedConnPool.Empty()
	}
}

// Stop triggers the graceful shutdown of all worker-routines.
func (tq *Taskqueue) Stop() {
	close(tq.quit)
}

// queueWorker connects to Redis and listens to the Redis-list for the according config.Task.
// This routine gets entries from Redis, tries to parse them into queueTask and sends them
// to the according instances of taskWorker.
func (tq *Taskqueue) queueWorker(ct config.Task, queue chan queueTask) {
	rc, err := redis.Dial(tq.config.RedisNetwork, tq.config.RedisAddress)
	if err != nil {
		tq.output.StopError(fmt.Sprintf("redis.Dial(): %s", err))
	}
	defer rc.Close()

	queueKey := tq.config.RedisQueueKey + ":" + ct.Type

	// This go-routine waits for the quit-channel to close, which signals to shutdown of
	// all worker-routines. We achieve that by closing the Redis-connection and catching that error.
	shutdown := false
	go func() {
		_, ok := <-tq.quit
		if !ok {
			shutdown = true
			rc.Close()
			tq.output.Debug(fmt.Sprintf("Shutting down workers for type %s", ct.Type))
		}
	}()

	for {
		values, err := rc.Cmd("BLPOP", queueKey, 0).List()
		if err != nil {
			// Errors here will likely be connection errors. Therefore we'll just
			// notify about the error and break the loop, which will stop the queueWorker
			// and all related taskWorker instances for this config.Task.
			// When shutdown == true, we're currently handling a graceful shutdown,
			// so we won't notify in that case and just break the loop.
			if shutdown == false {
				msg := fmt.Sprintf("Redis Error:\n%s\nStopping task %s.", err, ct.Type)
				tq.output.NotifyError(msg)
			}
			break
		}

		for _, value := range values {
			// BLPOP can return entries from multiple lists. It therefore includes the
			// list-name where the returned entry comes from, which we don't need, as we only have one list.
			// We only need the "real" entry, so we just skip that "value" from Redis.
			if value == queueKey {
				continue
			}

			tq.output.Debug(fmt.Sprintf("Task from redis for type %s with payload %s", ct.Type, value))

			task, err := parseQueueTask(value)
			if err != nil {
				// Errors from parseQueueTask will just result in a notification.
				// So we'll just skip this entry/task and continue with the next one.
				msg := fmt.Sprintf("parseQueueTask(): %s", err)
				tq.output.NotifyError(msg)
				continue
			}

			queue <- task
		}
	}

	close(queue)
	tq.waitGroup.Done()
}

// taskWorker waits for queueTask items and executes them. If they return an error,
// it the output object to notify about that error.
func (tq *Taskqueue) taskWorker(ct config.Task, queue chan queueTask) {
	for task := range queue {
		tq.output.Debug(fmt.Sprintf("Executing task type %s with payload %s", ct.Type, task.Args))
		tq.stats.IncrTaskCount(ct.Type)

		err := task.execute(ct.Script)

		if err != nil {
			task.ErrorMessage = fmt.Sprintf("%s", err)
			tq.addFailedTask(ct, task)

			msg := fmt.Sprintf("%s \"%s\"\n\n%s", ct.Script, strings.Join(task.Args, "\" \""), err)
			tq.output.NotifyError(msg)
		}
	}

	tq.waitGroup.Done()
}

// addFailedTask adds a failed task to a specific list into redis, so it can be handled
// afterwards. If the optional ttl-setting for these lists is not set, the feature is disabled.
func (tq *Taskqueue) addFailedTask(ct config.Task, qt queueTask) {
	if tq.config.FailedTasksTTL == 0 {
		return
	}

	rc, err := tq.failedConnPool.Get()
	if err != nil {
		tq.output.NotifyError(fmt.Sprintf("tq.failedConnPool.Get(): %s", err))
		return
	}

	queueKey := tq.config.RedisQueueKey + ":" + ct.Type + ":failed"

	jsonString, err := qt.getJsonString()
	if err != nil {
		tq.output.NotifyError(fmt.Sprintf("addFailedTask(), ct.getJsonString(): %s", err))
		return
	}

	// add to list
	reply := rc.Cmd("RPUSH", queueKey, jsonString)
	if reply.Err != nil {
		tq.output.NotifyError(fmt.Sprintf("addFailedTask(), RPUSH: %s", reply.Err))
		return
	}

	// set expire
	reply = rc.Cmd("EXPIRE", queueKey, tq.config.FailedTasksTTL)
	if reply.Err != nil {
		tq.output.NotifyError(fmt.Sprintf("addFailedTask(), EXPIRE: %s", reply.Err))
		return
	}

	tq.failedConnPool.Put(rc)
}