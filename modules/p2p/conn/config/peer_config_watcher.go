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
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-zookeeper/zk"
)

type PeerConfigWatcher struct {
	zkServers []string
	zkConn    *zk.Conn
	root      string
	onUpdate  func([]*PeerConfig)
	eventCh   <-chan zk.Event
}

func NewPeerConfigWatcher(zkServers []string, root string, onUpdate func([]*PeerConfig)) (*PeerConfigWatcher, error) {
	zkConn, _, err := zk.Connect(zkServers, 30*time.Second)
	if err != nil {
		panic(err)
	}

	exists, _, err := zkConn.Exists(root)
	if err != nil {
		return nil, err
	}

	if !exists {
		_, err := zkConn.Create(root, []byte{}, 0, zk.WorldACL(zk.PermAll))
		if err != nil && err != zk.ErrNodeExists {
			return nil, err
		}
	}

	_, _, eventCh, err := zkConn.ChildrenW(root)
	if err != nil {
		return nil, err
	}

	return &PeerConfigWatcher{
		zkServers: zkServers,
		zkConn:    zkConn,
		root:      root,
		onUpdate:  onUpdate,
		eventCh:   eventCh,
	}, nil
}

func (watcher *PeerConfigWatcher) Serve() {
	watcher.reloadPeerConfig()

	for {
		e := <-watcher.eventCh
		var err error
		_, _, watcher.eventCh, err = watcher.zkConn.ChildrenW(watcher.root)
		if err != nil {
			watcher.zkConn, _, err = zk.Connect(watcher.zkServers, 30*time.Second)
			if err != nil {
				panic(err)
			}

			_, _, watcher.eventCh, err = watcher.zkConn.ChildrenW(watcher.root)
			if err != nil {
				panic(err)
			}
			continue
		}

		if e.Type != zk.EventNodeChildrenChanged {
			fmt.Printf("[p2p_conn] Unknown event type got: %v\n", e.Type)
			time.Sleep(1 * time.Second)
		} else {
			fmt.Printf("new event: %v\n", e)
			watcher.reloadPeerConfig()
		}
	}
}

func (watcher *PeerConfigWatcher) reloadPeerConfig() {
	children, _, err := watcher.zkConn.Children(watcher.root)
	if err != nil {
		return
	}

	var configs []*PeerConfig
	for _, child := range children {
		fullpath := filepath.Join(watcher.root, child)
		data, _, err := watcher.zkConn.Get(fullpath)
		if err != nil {
			continue
		}

		fmt.Printf("\t%s: %s\n", fullpath, string(data))
		var cfg PeerConfig
		err = cfg.FromJsonStr(string(data))
		if err != nil {
			// TODO: log error
			continue
		}
		configs = append(configs, &cfg)
	}

	if watcher.onUpdate != nil {
		watcher.onUpdate(configs)
	}
}
