/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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
