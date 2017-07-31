// Package taskqueue provides the functionality for receiving, handling and executing tasks.
// In this file are the routines for the taskqueue itself.
package taskqueue

import (
	"fmt"
	"github.com/jpillora/backoff"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/nevsnode/gordon/config"
	"github.com/nevsnode/gordon/output"
	"github.com/nevsnode/gordon/stats"
	"sync"
	"time"
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

func init() {
	workerCount = make(map[string]int)
	workerBackoff = make(map[string]*backoff.Backoff)
}

// Start initialises several variables and creates necessary go-routines
func Start(c config.Config) {
	conf = c

	var err error
	redisPool, err = pool.NewCustom(conf.RedisNetwork, conf.RedisAddress, 0, redisDialFunction)
	if err != nil {
		output.NotifyError("redis pool.NewCustom():", err)
	}

	stats.InitTasks(conf.Tasks)

	failedChan = make(chan failedTask)
	shutdownChan = make(chan bool, 1)

	for _, ct := range conf.Tasks {
		createWorkerCount(ct.Type)
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
}

// Wait waits, to keep the application running as long as there are workers
func Wait() {
	waitGroup.Wait()
	output.Debug("Finished task-workers")

	close(failedChan)
	waitGroupFailed.Wait()
	output.Debug("Finished failed-task-worker")
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

intervalLoop:
	for <-runIntervalLoop {
		for taskType, configTask := range conf.Tasks {
			if isShuttingDown() {
				break intervalLoop
			}

			output.Debug("Checking for new tasks (" + taskType + ")")

			// check if there are available workers
			if !isWorkerAvailable(taskType) {
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
				if !isWorkerAvailable(taskType) {
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

				// spawn worker go-routine
				claimWorker(taskType)
				go taskWorker(task, configTask)

				// we've actually are handling new tasks so reset the interval
				interval.Reset()
			}
		}

		doneIntervalLoop <- true
	}

	Stop()
	waitGroup.Done()
	output.Debug("Finished queue-worker")
}

func taskWorker(task QueueTask, ct config.Task) {
	defer returnWorker(ct.Type)

	var errorBackoff *backoff.Backoff
	if ct.BackoffEnabled {
		errorBackoff = getBackoff(ct.Type)
	}
	if errorBackoff != nil {
		errorBackoff.Duration()
	}

	payload, _ := task.GetJSONString()
	output.Debug("Executing task type", ct.Type, "- Payload:", payload)
	txn := stats.StartedTask(ct.Type)

	err := task.Execute(ct.Script)

	if err == nil {
		resetBackoff(errorBackoff)
	}

	if err != nil {
		txn.NoticeError(err)
	}
	txn.End()

	if err != nil {
		task.ErrorMessage = fmt.Sprintf("%s", err)
		failedChan <- failedTask{
			configTask: ct,
			queueTask:  task,
		}

		msg := fmt.Sprintf("Failed executing task for type \"%s\"\nPayload:\n%s\n\n%s", ct.Type, payload, err)
		output.NotifyError(msg)
	}

	output.Debug("Finished task type", ct.Type, "- Payload:", payload)
}

func failedTaskWorker() {
	defer waitGroupFailed.Done()

	for ft := range failedChan {
		ct := ft.configTask
		qt := ft.queueTask

		if ct.FailedTasksTTL == 0 {
			continue
		}

		rc, err := redisPool.Get()
		if err != nil {
			output.NotifyError("redisPool.Get():", err)
			continue
		}
		defer redisPool.Put(rc)

		queueKey := conf.RedisQueueKey + ":" + ct.Type + ":failed"

		jsonString, err := qt.GetJSONString()
		if err != nil {
			output.NotifyError("failedTaskWorker(), qt.GetJSONString():", err)
			continue
		}

		// add to list
		reply := rc.Cmd("RPUSH", queueKey, jsonString)
		if reply.Err != nil {
			output.NotifyError("failedTaskWorker(), RPUSH:", reply.Err)
			continue
		}

		// set expire
		reply = rc.Cmd("EXPIRE", queueKey, ct.FailedTasksTTL)
		if reply.Err != nil {
			output.NotifyError("failedTaskWorker(), EXPIRE:", reply.Err)
			continue
		}
	}
}

func redisDialFunction(network, addr string) (*redis.Client, error) {
	return redis.DialTimeout(network, addr, time.Duration(10)*time.Second)
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
	defer shutdownLock.Unlock()
	shutdown = true
}

var (
	workerCount     map[string]int
	workerCountLock sync.Mutex
)

func createWorkerCount(taskType string) {
	workerCountLock.Lock()
	defer workerCountLock.Unlock()

	workerCount[taskType] = 0
}

func isWorkerAvailable(taskType string) bool {
	workerCountLock.Lock()
	defer workerCountLock.Unlock()

	currentCount := workerCount[taskType]
	maxCount := conf.Tasks[taskType].Workers

	return currentCount < maxCount
}

func claimWorker(taskType string) {
	waitGroup.Add(1)

	workerCountLock.Lock()
	defer workerCountLock.Unlock()

	workerCount[taskType]++
}

func returnWorker(taskType string) {
	workerCountLock.Lock()
	defer workerCountLock.Unlock()

	workerCount[taskType]--
	waitGroup.Done()
}

var (
	workerBackoff     map[string]*backoff.Backoff
	workerBackoffLock sync.Mutex
)

func getBackoff(taskType string) *backoff.Backoff {
	workerBackoffLock.Lock()
	defer workerBackoffLock.Unlock()

	if workerBackoff[taskType] == nil {
		ct := conf.Tasks[taskType]
		workerBackoff[taskType] = &backoff.Backoff{
			Min:    time.Duration(ct.BackoffMin) * time.Millisecond,
			Max:    time.Duration(ct.BackoffMax) * time.Millisecond,
			Factor: ct.BackoffFactor,
			Jitter: true,
		}
	}

	return workerBackoff[taskType]
}

func resetBackoff(bkoff *backoff.Backoff) {
	if bkoff == nil {
		return
	}

	workerBackoffLock.Lock()
	defer workerBackoffLock.Unlock()

	bkoff.Reset()
}
