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

package peer

import (
	"fmt"
	"net"
	"sync"

	"github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/arcology-network/main/modules/p2p/conn/protocol"
	"github.com/arcology-network/main/modules/p2p/conn/status"
)

type Switch struct {
	id            string
	plock         sync.RWMutex
	peers         map[string]*Peer
	onMsgReceived func(string, *protocol.Message)
	onPeerClosed  func(string)
	collector     *status.Collector
}

func NewSwitch(id string, onMsgReceived func(string, *protocol.Message), onPeerClosed func(string), collector *status.Collector) *Switch {
	return &Switch{
		id:            id,
		peers:         make(map[string]*Peer),
		onMsgReceived: onMsgReceived,
		onPeerClosed:  onPeerClosed,
		collector:     collector,
	}
}

func (sw *Switch) AddConnection(cfg *config.PeerConfig, conn net.Conn) {
	sw.plock.Lock()
	defer sw.plock.Unlock()

	if _, ok := sw.peers[cfg.Host]; !ok {
		sw.peers[cfg.Host] = NewPeer(sw.id, cfg, sw.onMsgReceived)
		go sw.peers[cfg.Host].Serve(func() {
			sw.plock.Lock()
			defer sw.plock.Unlock()

			delete(sw.peers, cfg.Host)
			sw.collector.Notify(status.Event{
				RemoteAddr: cfg.Host,
				Type:       status.EventTypePeerClosed,
				Data:       cfg,
			})
			fmt.Printf("Peer %s closed\n", cfg.Host)
			sw.onPeerClosed(cfg.ID)
		})
	}
	sw.peers[cfg.Host].AddConnection(conn)
	sw.collector.Notify(status.Event{
		RemoteAddr: conn.RemoteAddr().String(),
		Type:       status.EventTypeNewConnection,
		Data:       cfg,
	})
}

func (sw *Switch) GetPeerReady(host string) {
	sw.plock.Lock()
	defer sw.plock.Unlock()

	if _, ok := sw.peers[host]; ok {
		sw.peers[host].GetReady()
	} else {
		panic("peer not found")
	}
}

func (sw *Switch) Broadcast(msg *protocol.Message, pushIn bool) {
	sw.plock.RLock()
	defer sw.plock.RUnlock()

	fmt.Printf("[Switch.Broadcast] broadcast message to peers\n")
	for _, peer := range sw.peers {
		if !peer.IsReady() {
			fmt.Printf("[Switch.Broadcast] peer is not ready: %v\n", peer)
		}
		fmt.Printf("[Switch.Broadcast] peer: %v\n", peer.cfg)
		peer.Send(msg, pushIn)
	}
}

func (sw *Switch) Send(peer string, msg *protocol.Message) {
	sw.plock.RLock()
	defer sw.plock.RUnlock()

	fmt.Printf("[Switch.Send] send message to peer: %v\n", peer)
	// FIXME
	for _, p := range sw.peers {
		if p.cfg.ID == peer && p.IsReady() {
			p.Send(msg, true)
			return
		}
	}
	fmt.Printf("[Switch.Send] peer is not ready: %v\n", peer)
}
