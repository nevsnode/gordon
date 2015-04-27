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
	"github.com/fzzy/radix/redis"
	"os/exec"
	"strings"
	"sync"
)

// A QueueTask is the task as it is enqueued in a Redis-list.
type QueueTask struct {
	Args []string // list of arguments passed to the defined script/application
}

// execute executes the passed script/application with the arguments from the QueueTask object.
func (q QueueTask) execute(script string) error {
	out, err := exec.Command(script, q.Args...).Output()

	if len(out) != 0 && err == nil {
		err = fmt.Errorf("%s", out)
	}

	return err
}

// parseQueueTask parses the string-value from a Redis-list entry.
// It returns an object of QueueTask and a possible error if parsing failed.
func parseQueueTask(value string) (task QueueTask, err error) {
	reader := strings.NewReader(value)
	parser := json.NewDecoder(reader)
	err = parser.Decode(&task)
	return
}

// A Taskqueue offers routines for the task-workers and queue-workers.
// It also offers routines to set the config-, output- and stats-objects,
// which are used from the worker-routines.
type Taskqueue struct {
	WaitGroup sync.WaitGroup // wait group used to handle a proper application shutdown, in case of any Redis (connection) errors
	config    config.Config  // config object, storing, for instance, connection data
	output    output.Output  // output object used to write debug-/error-messages, mostly for notifying about errors on task-execution
	stats     *stats.Stats   // stats object for gathering usage data
}

// New returns a new instance of a Taskqueue
func New() Taskqueue {
	return Taskqueue{}
}

// SetConfig sets the config object
func (tq *Taskqueue) SetConfig(c config.Config) {
	tq.config = c
}

// SetOutput sets the output object
func (tq *Taskqueue) SetOutput(o output.Output) {
	tq.output = o
}

// SetStats sets the stats object
func (tq *Taskqueue) SetStats(s *stats.Stats) {
	tq.stats = s
}

// QueueWorker connects to Redis and listens to the Redis-list for the according config.Task.
// This routine gets entries from Redis, tries to parse them into QueueTask and sends them
// to the according instances of TaskWorker.
func (tq Taskqueue) QueueWorker(ct config.Task, queue chan QueueTask) {
	rc, err := redis.Dial(tq.config.RedisNetwork, tq.config.RedisAddress)
	if err != nil {
		tq.output.StopError(fmt.Sprintf("redis.Dial(): %s", err))
	}
	defer rc.Close()

	queueKey := tq.config.RedisQueueKey + ":" + ct.Type

	for {
		values, err := rc.Cmd("BLPOP", queueKey, 0).List()
		if err != nil {
			// Errors here will likely be connection errors. Therefore we'll just
			// notify about the error and break the loop, which will stop the QueueWorker
			// and all related TaskWorker instances for this config.Task.
			msg := fmt.Sprintf("Redis Error:\n%s\nStopping task %s.", err, ct.Type)
			tq.output.NotifyError(msg)
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
	tq.WaitGroup.Done()
}

// TaskWorker waits for QueueTask items and executes them. If they return an error,
// it the output object to notify about that error.
func (tq Taskqueue) TaskWorker(ct config.Task, queue chan QueueTask) {
	for task := range queue {
		tq.output.Debug(fmt.Sprintf("Executing task type %s with payload %s", ct.Type, task.Args))
		tq.stats.IncrTaskCount(ct.Type)

		err := task.execute(ct.Script)

		if err != nil {
			msg := fmt.Sprintf("%s %s\n\n%s", ct.Script, strings.Join(task.Args, " "), err)
			tq.output.NotifyError(msg)
		}
	}

	tq.WaitGroup.Done()
}
