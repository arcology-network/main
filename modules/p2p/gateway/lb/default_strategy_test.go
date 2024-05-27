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
