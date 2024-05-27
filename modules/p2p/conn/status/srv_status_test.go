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

package status

import (
	"testing"

	"github.com/arcology-network/main/modules/p2p/conn/config"
)

func TestToJson(t *testing.T) {
	status := SvcStatus{
		SvcConfig: &config.Config{
			Server: struct {
				NID  string `yaml:"nid" json:"nid"`
				SID  string `yaml:"sid" json:"sid"`
				Host string `yaml:"host" json:"host"`
				Port int    `yaml:"port" json:"port"`
			}{
				NID:  "node1",
				SID:  "svc1",
				Host: "127.0.0.1",
				Port: 9292,
			},
			ZooKeeper: struct {
				Servers        []string `yaml:"servers" json:"servers"`
				PeerConfigRoot string   `yaml:"croot" json:"croot"`
				ConnStatusRoot string   `yaml:"sroot" json:"sroot"`
			}{
				Servers:        []string{"localhost:2181"},
				PeerConfigRoot: "/p2p/peer/config",
				ConnStatusRoot: "/p2p/conn/status",
			},
			Kafka: struct {
				Servers  []string `yaml:"servers" json:"servers"`
				TopicIn  string   `yaml:"tin" json:"tin"`
				TopicOut string   `yaml:"tout" json:"tout"`
			}{
				Servers:  []string{"localhost:9092"},
				TopicIn:  "p2p-in",
				TopicOut: "p2p-out",
			},
		},
		Peers: map[string]*Peer{
			"peer1": {
				ID: "peer1",
				Config: &config.PeerConfig{
					ID:              "peer1",
					Host:            "localhost",
					Port:            9292,
					ConnectionCount: 2,
					AssignTo:        "me",
					Mode:            1,
				},
				Connections: []string{"conn1", "conn2"},
			},
		},
	}
	jsonStr := `{"config":{"server":{"nid":"node1","sid":"svc1","host":"127.0.0.1","port":9292},"zk":{"servers":["localhost:2181"],"croot":"/p2p/peer/config","sroot":"/p2p/conn/status"},"kafka":{"servers":["localhost:9092"],"tin":"p2p-in","tout":"p2p-out"}},"peers":{"peer1":{"id":"peer1","config":{"id":"peer1","host":"localhost","port":9292,"connections":2,"assign_to":"me","mode":1},"connections":["conn1","conn2"]}}}`

	json, err := status.ToJsonStr()
	if err != nil {
		t.Error(err)
		return
	}

	if json != jsonStr {
		t.Error("Fail")
	}
}

func TestFromJson(t *testing.T) {
	jsonStr := `{"config":{"server":{"nid":"node1","sid":"svc1","host":"127.0.0.1","port":9292},"zk":{"servers":["localhost:2181"],"croot":"/p2p/peer/config","sroot":"/p2p/conn/status"},"kafka":{"servers":["localhost:9092"],"tin":"p2p-in","tout":"p2p-out"}},"peers":{"peer1":{"id":"peer1","config":{"id":"peer1","host":"localhost","port":9292,"connections":2,"assign_to":"me","mode":1},"connections":["conn1","conn2"]}}}`

	var status SvcStatus
	err := status.FromJsonStr(jsonStr)
	if err != nil {
		t.Error(err)
		return
	}

	if status.SvcConfig.Server.NID != "node1" ||
		status.SvcConfig.Server.SID != "svc1" ||
		status.SvcConfig.Server.Host != "127.0.0.1" ||
		status.SvcConfig.Server.Port != 9292 ||
		len(status.SvcConfig.ZooKeeper.Servers) != 1 ||
		status.SvcConfig.ZooKeeper.Servers[0] != "localhost:2181" ||
		status.SvcConfig.ZooKeeper.PeerConfigRoot != "/p2p/peer/config" ||
		status.SvcConfig.ZooKeeper.ConnStatusRoot != "/p2p/conn/status" ||
		len(status.SvcConfig.Kafka.Servers) != 1 ||
		status.SvcConfig.Kafka.Servers[0] != "localhost:9092" ||
		status.SvcConfig.Kafka.TopicIn != "p2p-in" ||
		status.SvcConfig.Kafka.TopicOut != "p2p-out" ||
		len(status.Peers) != 1 {
		t.Error("Fail")
		return
	}

	if p, ok := status.Peers["peer1"]; !ok ||
		p.ID != "peer1" ||
		p.Config.ID != "peer1" ||
		p.Config.Host != "localhost" ||
		p.Config.Port != 9292 ||
		p.Config.ConnectionCount != 2 ||
		p.Config.AssignTo != "me" ||
		p.Config.Mode != 1 ||
		len(p.Connections) != 2 ||
		p.Connections[0] != "conn1" ||
		p.Connections[1] != "conn2" {
		t.Error("Fail")
	}
}
