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
	shutdownChan    chan bool
	waitGroup       sync.WaitGroup
	waitGroupFailed sync.WaitGroup
	redisPool       *pool.Pool
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

	failedChan = make(chan failedTask)
	shutdownChan = make(chan bool, 1)

	for _, ct := range conf.Tasks {
		var eb *backoff.Backoff
		if ct.BackoffEnabled {
			eb = &backoff.Backoff{
				Min:    time.Duration(ct.BackoffMin) * time.Millisecond,
				Max:    time.Duration(ct.BackoffMax) * time.Millisecond,
				Factor: ct.BackoffFactor,
				Jitter: true,
			}
		}

		createWorkerChan(ct.Type)

		for i := 0; i < ct.Workers; i++ {
			waitGroup.Add(1)
			go taskWorker(ct, eb)
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
	if isShuttingDown() {
		return
	}

	setShutdown()
	shutdownChan <- true

	closeWorkerChans()
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
	doneIntervalLoop := make(chan bool)

	go func() {
		for {
			<-doneIntervalLoop
			time.Sleep(interval.Duration())

			if isShuttingDown() {
				break
			}

			runIntervalLoop <- true
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

	doneIntervalLoop <- true

	for <-runIntervalLoop {
		for taskType, configTask := range conf.Tasks {
			if isShuttingDown() {
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

				sendWorkerTask(taskType, task)

				// we've actually are handling new tasks so reset the interval
				interval.Reset()
			}
		}

		doneIntervalLoop <- true
	}

	Stop()
	waitGroup.Done()
}

func taskWorker(ct config.Task, errorBackoff *backoff.Backoff) {
	wc := getWorkerChan(ct.Type)
	for task := range wc {
		output.Debug("Executing task type", ct.Type, "with arguments", task.Args)
		txn := stats.StartedTask(ct.Type)

		err := task.Execute(ct.Script)

		if txn != nil {
			if err != nil {
				txn.NoticeError(err)
			}
			txn.End()
		}

		if err != nil {
			task.ErrorMessage = fmt.Sprintf("%s", err)
			failedChan <- failedTask{
				configTask: ct,
				queueTask:  task,
			}

			msg := fmt.Sprintf("Failed executing task: %s \"%s\"\n%s", ct.Script, strings.Join(task.Args, "\" \""), err)
			output.NotifyError(msg)
		}

		if errorBackoff != nil {
			if err == nil {
				errorBackoff.Reset()
			} else {
				errorBackoff.Duration()
			}
		}

		output.Debug("Finished task type", ct.Type, "with arguments", task.Args)
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
	if isShuttingDown() {
		return false
	}

	return len(getWorkerChan(taskType)) < backlog
}

var (
	shutdownLock sync.RWMutex
	shutdown     = false
)

func isShuttingDown() bool {
	shutdownLock.RLock()
	defer shutdownLock.RUnlock()
	return shutdown
}

func setShutdown() {
	shutdownLock.Lock()
	shutdown = true
	shutdownLock.Unlock()
}

var (
	workerChanLock sync.Mutex
	workerChan     map[string]chan QueueTask
)

func createWorkerChan(taskType string) {
	workerChanLock.Lock()
	defer workerChanLock.Unlock()

	if workerChan == nil {
		workerChan = make(map[string]chan QueueTask)
	}

	workerChan[taskType] = make(chan QueueTask, backlog)
}

func getWorkerChan(taskType string) chan QueueTask {
	workerChanLock.Lock()
	defer workerChanLock.Unlock()

	return workerChan[taskType]
}

func sendWorkerTask(taskType string, task QueueTask) {
	workerChanLock.Lock()
	defer workerChanLock.Unlock()

	workerChan[taskType] <- task
}

func closeWorkerChans() {
	workerChanLock.Lock()
	defer workerChanLock.Unlock()

	for taskType := range workerChan {
		close(workerChan[taskType])
	}
}
