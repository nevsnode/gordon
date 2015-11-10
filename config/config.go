// Package config provides functionality to read and parse the Gordon configuration file.
package config

import (
	"github.com/BurntSushi/toml"
	"io/ioutil"
)

// A Config stores values, necessary for the execution of Gordon.
type Config struct {
	RedisNetwork   string      `toml:"redis_network"`    // network type used for the connection to Redis
	RedisAddress   string      `toml:"redis_address"`    // network address used for the connection to Redis
	RedisQueueKey  string      `toml:"queue_key"`        // first part of the list-names used in Redis
	ErrorScript    string      `toml:"error_script"`     // path to script/application that is executed when a task created an error
	FailedTasksTTL int         `toml:"failed_tasks_ttl"` // ttl for the lists that store failed tasks
	TempDir        string      `toml:"temp_dir"`         // path to a directory that is used for temporary files
	Logfile        string      // a file where all output will be written to, instead of stdout
	Stats          StatsConfig // options for the statistics package
	Tasks          []Task      // list of available tasks that Gordon can execute
}

type StatsConfig struct {
	Interface   string // the interface where statistics from Gordon can be gathered from
	Pattern     string // the pattern where the http-server will respond on
	TLSCertFile string `toml:"tls_cert_file"` // the certificate file used, to serve the statistics over https
	TLSKeyFile  string `toml:"tls_key_file"`  // the private key file used, to serve the statistics over https
}

// A Task stores information that task-workers need to execute their script/application.
type Task struct {
	Type    string // second part of the list-names used in Redis and used to identify tasks
	Script  string // path to the script/application that this task should execute
	Workers int    // number of concurrent go-routines, available for this task
}

// New reads the provided file and returns a Config instance with the values from it.
// It may also return an error, when the file doesn't exist, or the content could not be parsed.
func New(path string) (c Config, err error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	_, err = toml.Decode(string(file), &c)
	if err != nil {
		return
	}

	// take care of default-value
	if c.RedisNetwork == "" {
		c.RedisNetwork = "tcp"
	}

	return
}
