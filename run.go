package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/nightlyone/lockfile"
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
	Lockfile      string
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
	basepath, err := getBasePath()
	if err != nil {
		log.Fatal("getBasePath(): ", err)
	}

	config, err := getConfig(basepath)
	if err != nil {
		log.Fatal("getConfig(): ", err)
	}

	if !filepath.IsAbs(config.Lockfile) {
		config.Lockfile = basepath + "/" + config.Lockfile
	}
	lock, err := lockfile.New(config.Lockfile)
	if err != nil {
		log.Fatal("lockfile.New(): ", err)
	}
	err = lock.TryLock()
	if err != nil {
		if err == lockfile.ErrBusy {
			debugOutput(fmt.Sprintf("lock.TryLock(): %s", err))
			os.Exit(0)
		}
		log.Fatal("lock.TryLock(): ", err)
	}
	defer func() {
		err := lock.Unlock()
		if err != nil {
			log.Fatal("lock.Unlock(): ", err)
		}
	}()

	var wg sync.WaitGroup

	for _, task := range config.Tasks {
		debugOutput(fmt.Sprintf("Creating %d workers for type %s", task.Workers, task.Type))
		queue := make(chan QueueTaskStruct)

		for i := 0; i < task.Workers; i++ {
			wg.Add(1)
			go taskWorker(config, task, queue)
			debugOutput("Started worker for type " + task.Type)
		}

		go taskQueueWorker(config, task, queue)
		debugOutput("Started queue worker for type " + task.Type)
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

func getConfig(basepath string) (c ConfigStruct, err error) {
	file, err := os.Open(basepath + "/config.json")
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
