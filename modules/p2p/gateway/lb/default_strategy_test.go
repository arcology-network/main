package lb

import (
	"testing"

	"github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/arcology-network/main/modules/p2p/conn/status"
)

func TestDefaultStrategy(t *testing.T) {
	slaves := map[string]*status.SvcStatus{
		"svc1": {
			SvcConfig: &config.Config{
				Server: struct {
					NID  string `yaml:"nid" json:"nid"`
					SID  string `yaml:"sid" json:"sid"`
					Host string `yaml:"host" json:"host"`
					Port int    `yaml:"port" json:"port"`
				}{
					SID:  "svc1",
					Host: "127.0.0.1",
					Port: 9292,
				},
			},
			Peers: map[string]*status.Peer{
				"peer1": {
					Connections: make([]string, 4),
				},
				"peer2": {
					Connections: make([]string, 4),
				},
			},
		},
		"svc2": {
			SvcConfig: &config.Config{
				Server: struct {
					NID  string `yaml:"nid" json:"nid"`
					SID  string `yaml:"sid" json:"sid"`
					Host string `yaml:"host" json:"host"`
					Port int    `yaml:"port" json:"port"`
				}{
					SID:  "svc2",
					Host: "127.0.0.1",
					Port: 9293,
				},
			},
			Peers: map[string]*status.Peer{
				"peer1": {
					Connections: make([]string, 2),
				},
				"peer2": {
					Connections: make([]string, 4),
				},
			},
		},
		"svc3": {
			SvcConfig: &config.Config{
				Server: struct {
					NID  string `yaml:"nid" json:"nid"`
					SID  string `yaml:"sid" json:"sid"`
					Host string `yaml:"host" json:"host"`
					Port int    `yaml:"port" json:"port"`
				}{
					SID:  "svc3",
					Host: "127.0.0.1",
					Port: 9294,
				},
			},
			Peers: map[string]*status.Peer{
				"peer1": {
					Connections: make([]string, 8),
				},
				"peer2": {
					Connections: make([]string, 4),
				},
			},
		},
	}
	svcID := NewDefaultStrategy().GetSvcID(slaves, "peer", 0)
	if svcID != "svc2" {
		t.Error("Fail")
	}
}
