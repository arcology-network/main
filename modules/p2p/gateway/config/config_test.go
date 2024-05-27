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

	if cfg.Server.ID != "node1" ||
		cfg.Server.Host != "127.0.0.1" ||
		cfg.Server.Port != 9191 ||
		len(cfg.ZooKeeper.Servers) != 1 ||
		cfg.ZooKeeper.Servers[0] != "localhost:2181" ||
		cfg.ZooKeeper.PeerConfigRoot != "/p2p/peer/config" ||
		cfg.ZooKeeper.ConnStatusRoot != "/p2p/conn/status" {
		t.Error("Fail")
		return
	}
	if len(cfg.Peers) != 1 ||
		cfg.Peers[0].ID != "node2" ||
		cfg.Peers[0].Host != "127.0.0.1" ||
		cfg.Peers[0].Port != 9192 {
		t.Error("Fail")
		return
	}
}
