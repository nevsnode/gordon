package config

import (
	"testing"
)

func TestConfig(t *testing.T) {
	configFile := "./doesnotexist"
	_, _, err := New(configFile)
	if err == nil {
		t.Log("New() must return an error when file does not exist")
		t.FailNow()
	}

	configFile = "../example.gordon.config.toml"
	conf, _, err := New(configFile)
	if err != nil {
		t.Log("New() should not return an error when file exists and has a valid content")
		t.Log("err: ", err)
		t.Fail()
	}

	if conf.RedisNetwork != "tcp" && conf.RedisNetwork != "udp" {
		t.Log("RedisNetwork should always have a valid value")
		t.Fail()
	}

	if len(conf.Tasks) > 0 {
		for _, task := range conf.Tasks {
			if conf.BackoffEnabled && !task.IgnoreGlobalParams && !task.BackoffEnabled {
				t.Log("BackoffEnabled for a task should be true, when the global value is")
				t.Fail()
				break
			}

			if conf.BackoffEnabled && task.IgnoreGlobalParams && task.BackoffEnabled {
				t.Log("BackoffEnabled for a task should be false, when the global value is true but IgnoreGlobalParams is enabled")
				t.Fail()
				break
			}
		}

	}
}
