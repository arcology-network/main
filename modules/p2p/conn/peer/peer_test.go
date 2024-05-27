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

package peer

import (
	"fmt"
	"testing"
	"time"

	"github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/arcology-network/main/modules/p2p/conn/protocol"
)

func TestPeerSendRecv(t *testing.T) {
	cfg := &config.PeerConfig{
		Host:            "localhost",
		Port:            9292,
		ConnectionCount: 4,
	}

	count := 0
	peer := NewPeer("client", cfg, func(peerID string, m *protocol.Message) {
		count++
		if count%100000 == 0 {
			fmt.Printf("%d messages received\n", count)
		}
	})
	connections, err := peer.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	for _, c := range connections {
		peer.AddConnection(c.GetConn())
	}
	go peer.Serve(func() {
		t.Log("Peer closed")
	})

	start := time.Now()
	for i := 0; i < 1000000; i++ {
		peer.Send(&protocol.Message{
			Type: protocol.MessageTypeTestData,
			Data: make([]byte, 8000),
		}, false)
	}
	t.Log(time.Since(start))
	time.Sleep(3 * time.Second)
	peer.Disconnect()
}
