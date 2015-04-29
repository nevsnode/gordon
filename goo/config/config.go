// Package config provides functionality to read and parse the Goophry configuration file.
package config

import (
	"encoding/json"
	"os"
)

// A Config stores values, necessary for the execution of Goophry.
type Config struct {
	RedisNetwork   string // network type used for the connection to Redis
	RedisAddress   string // network address used for the connection to Redis
	RedisQueueKey  string // first part of the list-names used in Redis
	Tasks          []Task // list of available tasks that Goophry can execute
	ErrorCmd       string // command that will be executed when a taks created an error
	StatsInterface string // the interface where statistics from Goophry can be gathered from
	Logfile        string // a file where all output will be written to, instead of stdout
}

// A Task stores information that task-workers need to execute their script/application.
type Task struct {
	Type    string // second part of the list-names used in Redis and used to identify tasks
	Script  string // path to the script/application that this task should execute
	Workers int    // number of concurrent go-routines, available for this task
}

// NewConfig reads the provided file and returns a Config instance with the values from it.
// It may also return an error, when the file doesn't exist, or the content could not be parsed.
func NewConfig(path string) (c Config, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	parser := json.NewDecoder(file)
	err = parser.Decode(&c)
	return
}
