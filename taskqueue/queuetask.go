// Package taskqueue provides the functionality for receiving, handling and executing tasks.
// In this file are the routines for the task-structs used in the taskqueue.
package taskqueue

import (
	"encoding/json"
	"fmt"
	"github.com/nevsnode/gordon/config"
	"github.com/nevsnode/gordon/utils"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// A QueueTask is the task as it is enqueued in a Redis-list.
type QueueTask struct {
	Args         []string          `json:"args,omitempty"`          // list of arguments passed to script/application as argument in the given order
	Env          map[string]string `json:"env,omitempty"`           // map containing environment variables passed to script/application
	ErrorMessage string            `json:"error_message,omitempty"` // error message that might be created on executing the task
}

// Execute executes the passed script/application with the arguments from the QueueTask object.
func (q QueueTask) Execute(ct config.Task) (err error) {
	if ct.Webhook.IsSet() {
		err = doHTTPRequest(ct, q.Args, q.Env)
		if err != nil {
			return
		}
	}

	if ct.Script != "" {
		cmd := utils.ExecCommand(ct.Script, q.Args...)

		// add possible environment variables
		cmd.Env = os.Environ()
		for envKey, envVal := range q.Env {
			cmd.Env = append(cmd.Env, envKey+"="+envVal)
		}

		var out []byte
		out, err = cmd.Output()

		if err == nil && len(out) > 0 {
			err = fmt.Errorf("%s", out)
		}
	}

	return
}

// GetJSONString returns the QueueTask object as a json-encoded string
func (q QueueTask) GetJSONString() (value string, err error) {
	b, err := json.Marshal(q)
	value = fmt.Sprintf("%s", b)
	return
}

// NewQueueTask returns an instance of QueueTask from the passed value.
func NewQueueTask(redisValue string) (task QueueTask, err error) {
	if redisValue == "" {
		redisValue = "{}"
	}

	reader := strings.NewReader(redisValue)
	parser := json.NewDecoder(reader)
	err = parser.Decode(&task)

	if err == nil {
		// clear possible former error-message
		task.ErrorMessage = ""
	}

	return
}

var (
	httpClientLock sync.Mutex
	httpClients    map[string]*http.Client
)

// CreateHTTPClient creates the initial http-client used for webhook-requests
func CreateHTTPClient(ct config.Task) {
	httpClientLock.Lock()
	defer httpClientLock.Unlock()

	if httpClients == nil {
		httpClients = make(map[string]*http.Client)
	}

	httpClients[ct.Type] = &http.Client{
		Timeout: time.Second * time.Duration(ct.HTTPTimeout),
	}
}

func getHTTPClient(taskType string) *http.Client {
	httpClientLock.Lock()
	defer httpClientLock.Unlock()

	return httpClients[taskType]
}

const (
	envHeaderPrefix  = ":header:"
	envJSONKey       = ":json"
	envHTTPMethodKey = ":method"
)

func doHTTPRequest(ct config.Task, args []string, env map[string]string) (err error) {
	// build url
	webhookURL := strings.TrimRight(
		strings.TrimRight(
			ct.Webhook.URL,
			"/",
		)+"/"+strings.TrimLeft(
			strings.Join(args, "/"),
			"/",
		),
		"/",
	)

	// determine headers & parameters for request
	headers := make(map[string]string)
	for hKey, hVal := range ct.Webhook.Headers {
		headers[utils.PrepareHTTPHeader(hKey)] = hVal
	}
	params := make(map[string]string)
	for envKey, envVal := range env {
		if envKey == envJSONKey || envKey == envHTTPMethodKey {
			continue
		}

		if strings.HasPrefix(envKey, envHeaderPrefix) {
			envKey = strings.TrimPrefix(envKey, envHeaderPrefix)
			headers[utils.PrepareHTTPHeader(envKey)] = envVal
		} else {
			params[envKey] = envVal
		}
	}

	var req *http.Request

	method := ct.Webhook.Method
	if env[envHTTPMethodKey] != "" {
		env[envHTTPMethodKey] = strings.ToUpper(env[envHTTPMethodKey])
		if config.IsValidWebhookMethod(env[envHTTPMethodKey]) {
			method = env[envHTTPMethodKey]
		}
	}

	switch method {
	case "GET":
		// add query arguments to get-request
		req, err = http.NewRequest(method, webhookURL, nil)
		if err != nil {
			return
		}

		q := req.URL.Query()
		for key, value := range params {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
		break

	case "POST":
		if env[envJSONKey] != "" {
			// create rest-post-request
			req, err = http.NewRequest(method, webhookURL, strings.NewReader(env[envJSONKey]))
			if err != nil {
				return
			}

			headers["Content-Type"] = "application/json"
		} else {
			// otherwise a form-encoded request
			values := url.Values{}
			for key, value := range params {
				values.Add(key, value)
			}

			req, err = http.NewRequest(method, webhookURL, strings.NewReader(values.Encode()))
			if err != nil {
				return
			}

			headers["Content-Type"] = "application/x-www-form-urlencoded"
		}
		break
	}

	// attach headers to request
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// finally "do" request
	client := getHTTPClient(ct.Type)
	var res *http.Response
	res, err = client.Do(req)
	if err == nil {
		defer res.Body.Close()
		var body []byte
		body, err = ioutil.ReadAll(res.Body)

		if err == nil && len(body) > 0 {
			err = fmt.Errorf("%s", body)
		}
	}

	return
}
