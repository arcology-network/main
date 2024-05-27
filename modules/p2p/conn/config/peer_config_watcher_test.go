//go:build NOCI
// +build NOCI

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
	"time"

	"github.com/go-zookeeper/zk"
)

func TestPeerConfigWatcher(t *testing.T) {
	conn, _, err := zk.Connect([]string{"localhost:2181"}, 60*time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	watcher, err := NewPeerConfigWatcher(conn, "/p2p/peer/config", nil)
	if err != nil {
		t.Error(err)
		return
	}
	watcher.Serve()
}
