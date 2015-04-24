package config

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
    configFile := "./doesnotexist"
    _, err := New(configFile)
    assert.NotNil(t, err, "config must return an error, when file does not exist")

    configFile = "../../example.goophry.config.json"
    _, err = New(configFile)
    assert.Nil(t, err, "config should not return an error, when file exists and has valid json")
}
