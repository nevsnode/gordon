// Package taskqueue provides the functionality for receiving, handling and executing tasks.
// In this file are the routines for the taskqueue itself.
package taskqueue

import (
	"fmt"
	"github.com/jpillora/backoff"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/nevsnode/gordon/config"
	"github.com/nevsnode/gordon/output"
	"github.com/nevsnode/gordon/stats"
	"strings"
	"sync"
	"time"
)

const (
	backlog = 1
)

type failedTask struct {
	configTask config.Task
	queueTask  QueueTask
}

var (
	errorNoNewTask          = fmt.Errorf("No new task available")
	errorNoNewTasksAccepted = fmt.Errorf("No new tasks accepted")

	conf            config.Config
	shutdown        bool
	shutdownChan    chan bool
	waitGroup       sync.WaitGroup
	waitGroupFailed sync.WaitGroup
	redisPool       *pool.Pool
	errorBackoff    map[string]*backoff.Backoff
	workerChan      map[string]chan QueueTask
	failedChan      chan failedTask
)

// Start initialises several variables and creates necessary go-routines
func Start(c config.Config) {
	conf = c

	poolSize := 1
	for _, ct := range conf.Tasks {
		if ct.FailedTasksTTL > 0 {
			poolSize++
			break
		}
	}

	var err error
	redisPool, err = pool.New(conf.RedisNetwork, conf.RedisAddress, poolSize)
	if err != nil {
		output.NotifyError("redis pool.New():", err)
	}

	stats.InitTasks(conf.Tasks)

	workerChan = make(map[string]chan QueueTask)
	failedChan = make(chan failedTask)
	shutdownChan = make(chan bool, 1)

	errorBackoff = make(map[string]*backoff.Backoff)
	for _, ct := range conf.Tasks {
		if ct.BackoffEnabled {
			errorBackoff[ct.Type] = &backoff.Backoff{
				Min:    time.Duration(ct.BackoffMin) * time.Millisecond,
				Max:    time.Duration(ct.BackoffMax) * time.Millisecond,
				Factor: ct.BackoffFactor,
				Jitter: true,
			}
		}

		workerChan[ct.Type] = make(chan QueueTask, backlog)

		for i := 0; i < ct.Workers; i++ {
			waitGroup.Add(1)
			go taskWorker(ct)
		}
	}

	waitGroupFailed.Add(1)
	go failedTaskWorker()

	waitGroup.Add(1)
	go queueWorker()
}

// Stop will cause the taskqueue to stop accepting new tasks and shutdown the
// worker routines after they've finished their current tasks
func Stop() {
	if shutdown {
		return
	}

	shutdown = true
	shutdownChan <- true

	for taskType := range workerChan {
		close(workerChan[taskType])
	}
}

// Wait waits, to keep the application running as long as there are workers
func Wait() {
	waitGroup.Wait()

	close(failedChan)
	waitGroupFailed.Wait()

	redisPool.Empty()
}

func queueWorker() {
	interval := backoff.Backoff{
		Min:    time.Duration(conf.IntervalMin) * time.Millisecond,
		Max:    time.Duration(conf.IntervalMax) * time.Millisecond,
		Factor: conf.IntervalFactor,
	}

	runIntervalLoop := make(chan bool)
	var runningIntervalLoop sync.WaitGroup
	go func() {
		for {
			runningIntervalLoop.Wait()
			time.Sleep(interval.Duration())

			if shutdown {
				break
			}

			runIntervalLoop <- true
			runningIntervalLoop.Add(1)
		}
	}()

	go func() {
		for {
			select {
			case <-shutdownChan:
				runIntervalLoop <- false
			}
		}
	}()

	for <-runIntervalLoop {
		for taskType, configTask := range conf.Tasks {
			if shutdown {
				break
			}

			output.Debug("Checking for new tasks (" + taskType + ")")

			// check if there are available workers
			if !acceptsTasks(taskType) {
				continue
			}

			queueKey := conf.RedisQueueKey + ":" + taskType

			llen, err := redisPool.Cmd("LLEN", queueKey).Int()
			if err != nil {
				// Errors here are likely redis-connection errors, so we'll
				// need to notify about it
				output.NotifyError("redisPool.Cmd() Error:", err)
				break
			}

			// there are no new tasks in redis
			if llen == 0 {
				continue
			}

			// iterate over all entries in redis, until no more are available,
			// or all workers are busy, for a maximum of 2 * workers
			for i := 0; i < (configTask.Workers * 2); i++ {
				if !acceptsTasks(taskType) {
					break
				}

				value, err := redisPool.Cmd("LPOP", queueKey).Str()
				if err != nil {
					// no more tasks found
					break
				}

				output.Debug("Fetched task for type", taskType, "with payload", value)

				task, err := NewQueueTask(value)
				if err != nil {
					output.NotifyError("NewQueueTask():", err)
					continue
				}

				workerChan[taskType] <- task

				// we've actually are handling new tasks so reset the interval
				interval.Reset()
			}
		}

		runningIntervalLoop.Done()
	}

	Stop()
	waitGroup.Done()
}

func taskWorker(ct config.Task) {
	for task := range workerChan[ct.Type] {
		output.Debug("Executing task type", ct.Type, "with arguments", task.Args)
		stats.IncrTaskCount(ct.Type)

		err := task.Execute(ct.Script)
		if err != nil {
			task.ErrorMessage = fmt.Sprintf("%s", err)
			failedChan <- failedTask{
				configTask: ct,
				queueTask:  task,
			}

			msg := fmt.Sprintf("Failed executing task: %s \"%s\"\n%s", ct.Script, strings.Join(task.Args, "\" \""), err)
			output.NotifyError(msg)
		}

		if errorBackoff[ct.Type] != nil {
			if err == nil {
				errorBackoff[ct.Type].Reset()
			} else {
				time.Sleep(errorBackoff[ct.Type].Duration())
			}
		}
	}

	waitGroup.Done()
}

func failedTaskWorker() {
	for ft := range failedChan {
		ct := ft.configTask
		qt := ft.queueTask

		if ct.FailedTasksTTL == 0 {
			return
		}

		rc, err := redisPool.Get()
		if err != nil {
			output.NotifyError("redisPool.Get():", err)
			return
		}
		defer redisPool.Put(rc)

		queueKey := conf.RedisQueueKey + ":" + ct.Type + ":failed"

		jsonString, err := qt.GetJSONString()
		if err != nil {
			output.NotifyError("failedTaskWorker(), qt.GetJSONString():", err)
			return
		}

		// add to list
		reply := rc.Cmd("RPUSH", queueKey, jsonString)
		if reply.Err != nil {
			output.NotifyError("failedTaskWorker(), RPUSH:", reply.Err)
			return
		}

		// set expire
		reply = rc.Cmd("EXPIRE", queueKey, ct.FailedTasksTTL)
		if reply.Err != nil {
			output.NotifyError("failedTaskWorker(), EXPIRE:", reply.Err)
			return
		}
	}

	waitGroupFailed.Done()
}

func acceptsTasks(taskType string) bool {
	if shutdown {
		return false
	}

	return len(workerChan[taskType]) < backlog
}
