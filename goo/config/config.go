package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	RedisNetwork   string
	RedisAddress   string
	RedisQueueKey  string
	Tasks          []Task
	ErrorCmd       string
	StatsInterface string
	Logfile        string
}

type Task struct {
	Type    string
	Script  string
	Workers int
}

func New(path string) (c Config, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	parser := json.NewDecoder(file)
	err = parser.Decode(&c)
	return
}
