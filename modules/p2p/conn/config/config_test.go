package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig("config.yml")
	if err != nil {
		t.Error(err)
		return
	}

	if cfg.Server.NID != "node" {
		t.Errorf("Wrong nid: %s", cfg.Server.NID)
	}
	if cfg.Server.SID != "server" {
		t.Errorf("Wrong sid: %s", cfg.Server.SID)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("Wrong host: %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 9292 {
		t.Errorf("Wrong port: %d", cfg.Server.Port)
	}
	if len(cfg.ZooKeeper.Servers) != 1 || cfg.ZooKeeper.Servers[0] != "localhost:2181" {
		t.Errorf("Wrong zk servers: %v", cfg.ZooKeeper.Servers)
	}
	if cfg.ZooKeeper.PeerConfigRoot != "/p2p/peer/config" {
		t.Errorf("Wrong peer config root: %s", cfg.ZooKeeper.PeerConfigRoot)
	}
	if cfg.ZooKeeper.ConnStatusRoot != "/p2p/conn/status" {
		t.Errorf("Wrong conn status root: %s", cfg.ZooKeeper.ConnStatusRoot)
	}
	if len(cfg.Kafka.Servers) != 1 || cfg.Kafka.Servers[0] != "localhost:9092" {
		t.Errorf("Wrong kafka servers: %v", cfg.Kafka.Servers)
	}
	if cfg.Kafka.TopicIn != "p2p-in" {
		t.Errorf("Wrong topic in: %s", cfg.Kafka.TopicIn)
	}
	if cfg.Kafka.TopicOut != "p2p-out" {
		t.Errorf("Wrong topic out: %s", cfg.Kafka.TopicOut)
	}
}
