package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type ConfigStruct struct {
	RedisNetwork  string
	RedisAddress  string
	RedisQueueKey string
	Tasks         []TaskStruct
	ErrorCmd      string
}

type TaskStruct struct {
	Type    string
	Script  string
	Workers int
}

type QueueTaskStruct struct {
	Args []string
}

var verbose bool

func init() {
	flag.BoolVar(&verbose, "v", false, "enable verbose/debugging output")
	flag.Parse()
}

func main() {
	basepath := Basepath{}
	err := basepath.Update()
	if err != nil {
		log.Fatal("basepath.Update(): ", err)
	}

	config, err := getConfig(basepath.GetAbsWith("./goophry.config.json"))
	if err != nil {
		log.Fatal("getConfig(): ", err)
	}

	var wg sync.WaitGroup

	for _, task := range config.Tasks {
		debugOutput(fmt.Sprintf("Creating %d workers for type %s", task.Workers, task.Type))
		queue := make(chan QueueTaskStruct)

		task.Script = basepath.GetAbsWith(task.Script)

		for i := 0; i < task.Workers; i++ {
			wg.Add(1)
			go taskWorker(config, task, queue)
			debugOutput("Created worker for type " + task.Type)
		}

		go taskQueueWorker(config, task, queue)
		debugOutput("Created queue worker for type " + task.Type)
	}

	wg.Wait()
}

func debugOutput(msg string) {
	if verbose == true {
		log.Println(msg)
	}
}

func getBasePath() (path string, err error) {
	path, err = filepath.Abs(filepath.Dir(os.Args[0]))
	return
}

func getConfig(path string) (c ConfigStruct, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	parser := json.NewDecoder(file)
	err = parser.Decode(&c)
	return
}

func taskWorker(config ConfigStruct, task TaskStruct, queue chan QueueTaskStruct) {
	for t := range queue {
		debugOutput("Executing task for type " + task.Type)
		err := executeTask(task.Script, t.Args)
		if err != nil {
			errorCmdTask(config.ErrorCmd, task.Script, t.Args, err)
		}
	}
}

func taskQueueWorker(config ConfigStruct, task TaskStruct, queue chan QueueTaskStruct) {
	rc, err := redis.Dial(config.RedisNetwork, config.RedisAddress)
	if err != nil {
		log.Fatal("redis.Dial(): ", err)
	}
	defer rc.Close()

	queueKey := config.RedisQueueKey + ":" + task.Type

	for {
		values, err := redis.Strings(rc.Do("BLPOP", queueKey, 0))
		if err != nil {
			errorCmdRedis(config.ErrorCmd, err)
			continue
		}
		if len(values) == 0 {
			continue
		}

		for _, value := range values {
			if value == queueKey {
				continue
			}

			debugOutput("Task from redis: " + value)

			t, err := parseQueueTask(value)
			if err != nil {
				errorCmd(config.ErrorCmd, fmt.Sprintf("parseQueueTask(): ", err))
				continue
			}

			debugOutput("Sending task for type " + task.Type)
			queue <- t
		}
	}
}

func parseQueueTask(value string) (task QueueTaskStruct, err error) {
	reader := strings.NewReader(value)
	parser := json.NewDecoder(reader)
	err = parser.Decode(&task)
	return
}

func executeTask(script string, args []string) error {
	out, err := exec.Command(script, args...).Output()

	if len(out) != 0 && err == nil {
		err = fmt.Errorf("%s", out)
	}

	return err
}

func errorCmdTask(cmd string, script string, args []string, err error) {
	msg := fmt.Sprintf("%s %s\n\n%s", script, strings.Join(args, " "), err)
	errorCmd(cmd, msg)
}

func errorCmdRedis(cmd string, err error) {
	msg := fmt.Sprintf("Redis Error:\n%s", err)
	errorCmd(cmd, msg)
}

func errorCmd(cmd string, msg string) {
	debugOutput(fmt.Sprintf("Calling ErrorCmd with: %s", msg))

	cmdExec := fmt.Sprintf(cmd, strconv.Quote(msg))
	out, err := exec.Command("sh", "-c", cmdExec).Output()

	if len(out) != 0 && err == nil {
		log.Println(fmt.Sprintf("Error calling ErrorCmd:\n%s\n\nOutput:\n%s", cmdExec, out))
	}

	if err != nil {
		log.Println(fmt.Sprintf("Error calling ErrorCmd:\n%s", err))
	}
}

type Basepath struct {
	Path string
}

func (b *Basepath) Update() (err error) {
	b.Path, err = filepath.Abs(filepath.Dir(os.Args[0]))
	return
}

func (b Basepath) GetAbsWith(file string) (string) {
	if !filepath.IsAbs(file) {
		file = b.Path + "/" + file
	}
	return file
}
