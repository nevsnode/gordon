// Package taskqueue provides the functionality for receiving, handling and executing tasks.
// This package provides the routines for the task- and queue-workers.
// Queue-workers are the go-routines that wait for entries in the Redis-lists,
// parse them and send them to the task-workers.
// Task-workers are the go-routines that finally execute the tasks that they receive
// from the queue-workers.
// In this file are the routines for the task-structs used in the taskqueue.
package taskqueue

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// A QueueTask is the task as it is enqueued in a Redis-list.
type QueueTask struct {
	Args         []string `json:"args"`          // list of arguments passed to the defined script/application
	ErrorMessage string   `json:"error_message"` // error message that might be created on executing the task
}

// Execute executes the passed script/application with the arguments from the QueueTask object.
func (q QueueTask) Execute(script string) error {
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

// GetJSONString returns the QueueTask object as a json-encoded string
func (q QueueTask) GetJSONString() (value string, err error) {
	b, err := json.Marshal(q)
	value = fmt.Sprintf("%s", b)
	return
}

// NewQueueTask returns an instance of QueueTask from the passed value.
func NewQueueTask(redisValue string) (task QueueTask, err error) {
	reader := strings.NewReader(redisValue)
	parser := json.NewDecoder(reader)
	err = parser.Decode(&task)
	return
}
