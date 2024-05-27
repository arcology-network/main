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

package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/arcology-network/main/modules/p2p/conn/peer"
	"github.com/arcology-network/main/modules/p2p/conn/protocol"
)

func main() {
	host := os.Args[1]
	thread, _ := strconv.Atoi(os.Args[2])
	length, _ := strconv.Atoi(os.Args[3])
	cfg := &config.PeerConfig{
		Host:            host,
		Port:            9292,
		ConnectionCount: byte(thread),
	}

	count := 0
	peer := peer.NewPeer("client", cfg, func(peerID string, m *protocol.Message) {
		count++
		if count%100000 == 0 {
			fmt.Printf("%d messages received\n", count)
		}
	})
	connections, err := peer.Connect()
	if err != nil {
		return
	}
	for _, c := range connections {
		peer.AddConnection(c.GetConn())
	}
	go peer.Serve(func() {
		fmt.Printf("Peer closed\n")
	})

	start := time.Now()
	for i := 0; i < 1000000; i++ {
		peer.Send(&protocol.Message{
			Type: protocol.MessageTypeTestData,
			Data: make([]byte, length),
		}, false)
	}
	fmt.Printf("time elapsed %v\n", time.Since(start))
	time.Sleep(1 * time.Second)
	peer.Disconnect()
}
