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
	"bytes"
	"testing"

	"github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/arcology-network/main/modules/p2p/conn/connection"
	"github.com/arcology-network/main/modules/p2p/conn/mock"
	"github.com/arcology-network/main/modules/p2p/conn/protocol"
)

func TestReadSinglePackage(t *testing.T) {
	dataChan := make(chan *protocol.Message, 10)
	disconnected := make(chan struct{}, 10)
	reader := NewPeerReader(4, dataChan, disconnected)
	go reader.Serve()

	conn, _ := connection.NewConnection("MockConnection", &config.PeerConfig{}, mock.NewTCPConnection())
	reader.AddConnection(conn)

	m := protocol.Message{
		ID:   0xcc,
		Type: 0xdd,
		Data: []byte{0x11, 0x22, 0x33},
	}
	if err := protocol.WriteMessage(conn.GetConn(), &m); err != nil {
		t.Error(err)
		return
	}

	m2 := <-dataChan
	if m.ID != m2.ID ||
		m.Type != m2.Type ||
		!bytes.Equal(m.Data, m2.Data) {
		t.Error("Fail")
		return
	}
}

func TestReadMultiPackages(t *testing.T) {
	dataChan := make(chan *protocol.Message, 10)
	disconnected := make(chan struct{}, 10)
	reader := NewPeerReader(4, dataChan, disconnected)
	go reader.Serve()

	conn, _ := connection.NewConnection("MockConnection", &config.PeerConfig{}, mock.NewTCPConnection())
	reader.AddConnection(conn)

	data := make([]byte, protocol.MaxPackageBodySize*10)
	for i := 0; i < protocol.MaxPackageBodySize; i++ {
		data[i] = byte(i % 256)
	}
	m := protocol.Message{
		ID:   0xcc,
		Type: 0xdd,
		Data: data,
	}
	if err := protocol.WriteMessage(conn.GetConn(), &m); err != nil {
		t.Error(err)
		return
	}

	m2 := <-dataChan
	if m.ID != m2.ID ||
		m.Type != m2.Type ||
		!bytes.Equal(m.Data, m2.Data) {
		t.Error("Fail")
		return
	}
}
