// Package config provides functionality to read and parse the Gordon configuration file.
package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/nevsnode/gordon/utils"
	"io/ioutil"
	"strings"
)

// DefaultConfig describes the default path to the configuration file.
var defaultConfigs = []string{
	"./gordon.config.toml",
	"/etc/gordon.config.toml",
}

// A Config stores values, necessary for the execution of Gordon.
type Config struct {
	RedisNetwork   string          `toml:"redis_network"`    // network type used for the connection to Redis
	RedisAddress   string          `toml:"redis_address"`    // network address used for the connection to Redis
	RedisQueueKey  string          `toml:"queue_key"`        // first part of the list-names used in Redis
	ErrorScript    string          `toml:"error_script"`     // path to script/application that is executed when a task created an error
	ErrorWebhook   string          `toml:"error_webhook"`    // url to webhook that is called when a task created an error
	FailedTasksTTL int             `toml:"failed_tasks_ttl"` // ttl for the lists that store failed tasks
	TempDir        string          `toml:"temp_dir"`         // path to a directory that is used for temporary files
	IntervalMin    int             `toml:"interval_min"`     // minimum interval for checking for new tasks
	IntervalMax    int             `toml:"interval_max"`     // maxiumum interval for checking for new tasks
	IntervalFactor float64         `toml:"interval_factor"`  // multiplicator for the task-check interval
	BackoffEnabled bool            `toml:"backoff_enabled"`  // general flag to disable/enable error-backoff
	BackoffMin     int             `toml:"backoff_min"`      // general error-backoff start value in milliseconds
	BackoffMax     int             `toml:"backoff_max"`      // general error-backoff maximum value in milliseconds
	BackoffFactor  float64         `toml:"backoff_factor"`   // general error-backoff multiplicator
	HTTPTimeout    int             `toml:"http_timeout"`     // timeout for http-/webhook-requests
	WebhookMethod  string          `toml:"webhook_method"`   // http-method that is used for webhook-requests
	Logfile        string          // a file where all output will be written to, instead of stdout
	Stats          StatsConfig     // options for the statistics package
	Tasks          map[string]Task // map of available tasks that Gordon can execute
}

// StatsConfig contains configuration options for the stats-package/service.
type StatsConfig struct {
	Interface string         // the interface where statistics from Gordon can be gathered from
	NewRelic  NewRelicConfig // options for newrelic agent
}

// A Task stores information that task-workers need to execute their script/application.
type Task struct {
	Type                 string        // second part of the list-names used in Redis and used to identify tasks
	Script               string        // path to the script/application that this task should execute
	Workers              int           // number of concurrent go-routines available for this task
	IgnoreGlobalSettings bool          `toml:"ignore_global_settings"` // flag to ignore global settings
	ErrorScript          string        `toml:"error_script"`           // path to script/application that is executed when a task created an error
	ErrorWebhook         string        `toml:"error_webhook"`          // url to webhook that is called when a task created an error
	FailedTasksTTL       int           `toml:"failed_tasks_ttl"`       // ttl for the lists that store failed tasks
	BackoffEnabled       bool          `toml:"backoff_enabled"`        // task-specific flag to disable/enable error-backoff
	BackoffMin           int           `toml:"backoff_min"`            // task specific error-backoff start value in milliseconds
	BackoffMax           int           `toml:"backoff_max"`            // task specific error-backoff maximum value in milliseconds
	BackoffFactor        float64       `toml:"backoff_factor"`         // task specific error-backoff multiplicator
	HTTPTimeout          int           `toml:"http_timeout"`           // timeout for http-/webhook-requests
	Webhook              WebhookConfig // webhook configuration for this task
}

// A WebhookConfig holds information regarding a webhook that is called as a task.
type WebhookConfig struct {
	URL     string            // (base-)url of the task
	Method  string            // http-method that is used for the request
	Headers map[string]string // http-headers that are set for the request
}

// IsSet determines whether a webhook is configured or not
func (w WebhookConfig) IsSet() bool {
	return w.URL != ""
}

// NewRelicConfig stores information for the agent.
type NewRelicConfig struct {
	License string // the newrelic license key
	AppName string `toml:"app_name"` // the newrelic app-name
}

const (
	defaultRedisNetwork  = "tcp"
	minInterval          = 50
	defaultHTTPTimeout   = 60
	defaultWebhookMethod = "POST"
)

// New reads the provided file and returns a Config instance with the values from it.
// It may also return an error, when the file doesn't exist, or the content could not be parsed.
func New(path string) (c Config, cpath string, err error) {
	var (
		file    []byte
		configs []string
	)
	if path != "" {
		configs = append(configs, path)
	} else {
		configs = defaultConfigs
	}

	for _, cpath = range configs {
		file, err = ioutil.ReadFile(cpath)
		if err == nil {
			break
		}
	}
	if err != nil {
		return
	}

	_, err = toml.Decode(string(file), &c)
	if err != nil {
		return
	}

	// take care of default-values
	if c.RedisNetwork == "" {
		c.RedisNetwork = defaultRedisNetwork
	}
	if !isValidRedisNetwork(c.RedisNetwork) {
		err = fmt.Errorf("Invalid redis_network set")
		return
	}
	c.WebhookMethod = strings.ToUpper(c.WebhookMethod)
	if c.WebhookMethod == "" {
		c.WebhookMethod = defaultWebhookMethod
	}

	if !IsValidWebhookMethod(c.WebhookMethod) {
		err = fmt.Errorf("Invalid webhook_method set")
		return
	}

	// ensure reasonable values
	if c.IntervalMin < minInterval {
		c.IntervalMin = minInterval
	}
	if c.IntervalMax < c.IntervalMin {
		c.IntervalMax = c.IntervalMin
	}
	if c.IntervalFactor < 1 {
		c.IntervalFactor = 1
	}
	if c.BackoffMin < 0 {
		c.BackoffMin = 0
	}
	if c.BackoffMax < 0 {
		c.BackoffMax = 0
	}
	if c.BackoffFactor < 0 {
		c.BackoffFactor = 0
	}
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = defaultHTTPTimeout
	}

	for taskType, task := range c.Tasks {
		task.Type = taskType
		task.Script = utils.Basepath(task.Script)
		task.Webhook.Method = strings.ToUpper(task.Webhook.Method)

		if task.Workers < 1 {
			task.Workers = 1
		}

		if task.Webhook.Headers == nil {
			task.Webhook.Headers = make(map[string]string)
		}

		if task.IgnoreGlobalSettings == false {
			// use global error-script if not already set
			if task.ErrorScript == "" {
				task.ErrorScript = c.ErrorScript
			}

			// use global failed-task-ttl if not already set
			if task.FailedTasksTTL == 0 && c.FailedTasksTTL > 0 {
				task.FailedTasksTTL = c.FailedTasksTTL
			}

			// 'override' global settings unless customized
			if c.BackoffEnabled {
				task.BackoffEnabled = true
			}
			if task.BackoffMin == 0 {
				task.BackoffMin = c.BackoffMin
			}
			if task.BackoffMax == 0 {
				task.BackoffMax = c.BackoffMax
			}
			if task.BackoffFactor == 0 {
				task.BackoffFactor = c.BackoffFactor
			}
			if task.HTTPTimeout == 0 {
				task.HTTPTimeout = c.HTTPTimeout
			}
			if task.Webhook.Method == "" {
				task.Webhook.Method = c.WebhookMethod
			}
			if task.ErrorWebhook == "" {
				task.ErrorWebhook = c.ErrorWebhook
			}
		}

		// ensure reasonable values
		if task.BackoffMin < 0 {
			task.BackoffMin = 0
		}
		if task.BackoffMax < task.BackoffMin {
			task.BackoffMax = task.BackoffMin
		}
		if task.BackoffFactor < 1 {
			task.BackoffFactor = 1
		}
		if task.HTTPTimeout <= 0 {
			task.HTTPTimeout = defaultHTTPTimeout
		}
		if task.Webhook.Method == "" {
			task.Webhook.Method = defaultWebhookMethod
		}

		if !IsValidWebhookMethod(task.Webhook.Method) {
			err = fmt.Errorf("Invalid method for webhook in task %s", task.Type)
			return
		}

		c.Tasks[taskType] = task
	}

	return
}

func isValidRedisNetwork(rn string) bool {
	return rn == "tcp" || rn == "udp"
}

// IsValidWebhookMethod checks if the given value is a valid http method for a webhook
func IsValidWebhookMethod(method string) bool {
	return method == "GET" || method == "POST"
}
