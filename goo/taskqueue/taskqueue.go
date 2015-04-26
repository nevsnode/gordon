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

type QueueTask struct {
	Args []string
}

func (q QueueTask) execute(script string) error {
	out, err := exec.Command(script, q.Args...).Output()

	if len(out) != 0 && err == nil {
		err = fmt.Errorf("%s", out)
	}

	return err
}

func parseQueueTask(value string) (task QueueTask, err error) {
	reader := strings.NewReader(value)
	parser := json.NewDecoder(reader)
	err = parser.Decode(&task)
	return
}

type Taskqueue struct {
	WaitGroup sync.WaitGroup
	config    config.Config
	output    output.Output
	stats     *stats.Stats
}

func New() Taskqueue {
	return Taskqueue{}
}

func (tq *Taskqueue) SetConfig(c config.Config) {
	tq.config = c
}

func (tq *Taskqueue) SetOutput(o output.Output) {
	tq.output = o
}

func (tq *Taskqueue) SetStats(s *stats.Stats) {
	tq.stats = s
}

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
			msg := fmt.Sprintf("Redis Error:\n%s\nStopping task %s.", err, ct.Type)
			tq.output.NotifyError(msg)
			break
		}

		for _, value := range values {
			if value == queueKey {
				continue
			}

			tq.output.Debug(fmt.Sprintf("Task from redis for type %s with payload %s", ct.Type, value))

			task, err := parseQueueTask(value)
			if err != nil {
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
