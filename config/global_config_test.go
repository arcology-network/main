package config

import "testing"

func TestLoadGlobalConfig(t *testing.T) {
	config := LoadGlobalConfig("./global.json")
	t.Log(config)
}
