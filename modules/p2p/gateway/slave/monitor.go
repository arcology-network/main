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

package slave

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/arcology-network/main/modules/p2p/conn/status"
	"github.com/go-zookeeper/zk"
)

type Monitor struct {
	zkServers      []string
	zkConn         *zk.Conn
	root           string
	services       map[string]*status.SvcStatus
	onStatusUpdate func(map[string]*status.SvcStatus)
	eventCh        <-chan zk.Event
}

func NewMonitor(zkServers []string, root string, onStatusUpdate func(map[string]*status.SvcStatus)) *Monitor {
	zkConn, _, err := zk.Connect(zkServers, 30*time.Second)
	if err != nil {
		panic(err)
	}

	m := &Monitor{
		zkServers:      zkServers,
		zkConn:         zkConn,
		root:           root,
		services:       make(map[string]*status.SvcStatus),
		onStatusUpdate: onStatusUpdate,
	}

	_, _, m.eventCh, err = m.zkConn.ChildrenW(m.root)
	if err != nil {
		panic(err)
	}
	return m
}

func (m *Monitor) Serve() {
	m.reloadSrvStatus()
	m.onStatusUpdate(m.services)

	for {
		e := <-m.eventCh
		var err error
		_, _, m.eventCh, err = m.zkConn.ChildrenW(m.root)
		if err != nil {
			m.zkConn, _, err = zk.Connect(m.zkServers, 30*time.Second)
			if err != nil {
				panic(err)
			}

			_, _, m.eventCh, err = m.zkConn.ChildrenW(m.root)
			if err != nil {
				panic(err)
			}
			continue
		}

		if e.Type == zk.EventNodeChildrenChanged {
			m.reloadSrvStatus()
			m.onStatusUpdate(m.services)
		} else {
			fmt.Printf("[p2p_gateway] Unknown event type got %v\n", e.Type)
			time.Sleep(1 * time.Second)
		}
	}
}

func (m *Monitor) reloadSrvStatus() {
	m.services = make(map[string]*status.SvcStatus)
	children, _, err := m.zkConn.Children(m.root)
	if err != nil {
		panic(err)
	}

	for _, child := range children {
		fullpath := filepath.Join(m.root, child)
		data, _, err := m.zkConn.Get(fullpath)
		if err != nil {
			continue
		}

		var srvStatus status.SvcStatus
		err = srvStatus.FromJsonStr(string(data))
		if err != nil {
			continue
		}
		m.services[srvStatus.SvcConfig.Server.SID] = &srvStatus
	}
}
