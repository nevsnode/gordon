package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfig(t *testing.T) {
	configFile := "./doesnotexist"
	_, err := NewConfig(configFile)
	assert.NotNil(t, err, "config must return an error, when file does not exist")

	configFile = "../../example.gordon.config.json"
	conf, err := NewConfig(configFile)
	assert.Nil(t, err, "config should not return an error, when file exists and has valid json")

	assert.Contains(t, []string{"tcp", "udp"}, conf.RedisNetwork, "RedisNetwork should always have a (valid) value")
}
