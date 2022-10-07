package config

import "testing"

func TestLoadKafkaConfig(t *testing.T) {
	config := LoadKafkaConfig("./kafka.json")
	t.Log(config)
}
