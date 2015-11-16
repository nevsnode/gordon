package config

import (
	"testing"
)

func TestConfig(t *testing.T) {
	configFile := "./doesnotexist"
	_, err := New(configFile)
	if err == nil {
		t.Log("New() must return an error when file does not exist")
		t.FailNow()
	}

	configFile = "../example.gordon.config.toml"
	conf, err := New(configFile)
	if err != nil {
		t.Log("New() should not return an error when file exists and has a valid content")
		t.Log("err: ", err)
		t.Fail()
	}

	if conf.RedisNetwork != "tcp" && conf.RedisNetwork != "udp" {
		t.Log("RedisNetwork should always have a valid value")
		t.Fail()
	}
}
