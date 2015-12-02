// Package taskqueue provides the functionality for receiving, handling and executing tasks.
// This package provides the routines for the task- and queue-workers.
// Queue-workers are the go-routines that wait for entries in the Redis-lists,
// parse them and send them to the task-workers.
// Task-workers are the go-routines that finally execute the tasks that they receive
// from the queue-workers.
// In this file are the routines for the taskqueue itself.
package taskqueue

import (
	"../config"
	"../output"
	"../stats"
	"fmt"
	"github.com/jpillora/backoff"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"strings"
	"sync"
	"time"
)

// A Taskqueue offers routines for the task-workers and queue-workers.
// It also offers routines to set the config-, output- and stats-objects,
// which are used from the worker-routines.
type Taskqueue struct {
	waitGroup      sync.WaitGroup              // wait group used to handle a proper application shutdown
	config         config.Config               // config object, storing, for instance, connection data
	output         output.Output               // output object for handling debug-/error-messages and notifying about task execution errors
	stats          *stats.Stats                // stats object for gathering usage data
	quit           chan int                    // channel used to gracefully shutdown all go-routines
	failedConnPool *pool.Pool                  // pool of connections used for inserting failed task into their lists
	errorBackoff   map[string]*backoff.Backoff // map of backoff instances, each for every task-type
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

	// calculate size of connection-pool for the addFailedTask-routine
	poolSize := 0
	for _, configTask := range c.Tasks {
		if configTask.FailedTasksTTL > 0 {
			poolSize += configTask.Workers
		}
	}

	if poolSize > 0 {
		tq.failedConnPool, err = pool.New(c.RedisNetwork, c.RedisAddress, poolSize)
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

// createWorkers creates all worker go-routines.
func (tq *Taskqueue) createWorkers(ct config.Task) {
	queue := make(chan QueueTask)

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

// Start handles the creation of all workers for all configured tasks,
// and the initialization for the stats-package.
func (tq *Taskqueue) Start() {
	tq.errorBackoff = make(map[string]*backoff.Backoff, 0)

	for _, configTask := range tq.config.Tasks {
		if configTask.BackoffEnabled {
			tq.errorBackoff[configTask.Type] = &backoff.Backoff{
				Min:    time.Duration(configTask.BackoffMin) * time.Millisecond,
				Max:    time.Duration(configTask.BackoffMax) * time.Millisecond,
				Factor: configTask.BackoffFactor,
				Jitter: true,
			}
		}

		tq.stats.InitTask(configTask.Type)

		tq.createWorkers(configTask)
	}
}

// queueWorker connects to Redis and listens to the Redis-list for the according config.Task.
// This routine gets entries from Redis, tries to parse them into QueueTask and sends them
// to the according instances of taskWorker.
func (tq *Taskqueue) queueWorker(ct config.Task, queue chan QueueTask) {
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

			tq.output.Debug(fmt.Sprintf("Received task for type %s with payload %s", ct.Type, value))

			task, err := NewQueueTask(value)
			if err != nil {
				// Errors from NewQueueTask will just result in a notification.
				// So we'll just skip this entry/task and continue with the next one.
				msg := fmt.Sprintf("NewQueueTask(): %s", err)
				tq.output.NotifyError(msg)
				continue
			}

			queue <- task
		}
	}

	close(queue)
	tq.waitGroup.Done()
}

// taskWorker waits for QueueTask items and executes them. If they return an error,
// it the output object to notify about that error.
func (tq *Taskqueue) taskWorker(ct config.Task, queue chan QueueTask) {
	for task := range queue {
		tq.output.Debug(fmt.Sprintf("Executing task type %s with payload %s", ct.Type, task.Args))
		tq.stats.IncrTaskCount(ct.Type)

		err := task.Execute(ct.Script)

		if err != nil {
			task.ErrorMessage = fmt.Sprintf("%s", err)
			tq.addFailedTask(ct, task)

			msg := fmt.Sprintf("Failed executing task:\n%s \"%s\"\n\n%s", ct.Script, strings.Join(task.Args, "\" \""), err)
			tq.output.NotifyError(msg)
		}

		if tq.errorBackoff[ct.Type] != nil {
			if err == nil {
				tq.errorBackoff[ct.Type].Reset()
			} else {
				time.Sleep(tq.errorBackoff[ct.Type].Duration())
			}
		}
	}

	tq.waitGroup.Done()
}

// addFailedTask adds a failed task to a specific list into redis, so it can be handled
// afterwards. If the optional ttl-setting for these lists is not set, the feature is disabled.
func (tq *Taskqueue) addFailedTask(ct config.Task, qt QueueTask) {
	if ct.FailedTasksTTL == 0 {
		return
	}

	rc, err := tq.failedConnPool.Get()
	if err != nil {
		tq.output.NotifyError(fmt.Sprintf("tq.failedConnPool.Get(): %s", err))
		return
	}
	defer tq.failedConnPool.Put(rc)

	queueKey := tq.config.RedisQueueKey + ":" + ct.Type + ":failed"

	jsonString, err := qt.GetJSONString()
	if err != nil {
		tq.output.NotifyError(fmt.Sprintf("addFailedTask(), ct.GetJSONString(): %s", err))
		return
	}

	// add to list
	reply := rc.Cmd("RPUSH", queueKey, jsonString)
	if reply.Err != nil {
		tq.output.NotifyError(fmt.Sprintf("addFailedTask(), RPUSH: %s", reply.Err))
		return
	}

	// set expire
	reply = rc.Cmd("EXPIRE", queueKey, ct.FailedTasksTTL)
	if reply.Err != nil {
		tq.output.NotifyError(fmt.Sprintf("addFailedTask(), EXPIRE: %s", reply.Err))
		return
	}
}
