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
	"bufio"

	"github.com/arcology-network/main/modules/p2p/conn/connection"
	"github.com/arcology-network/main/modules/p2p/conn/protocol"
)

type PeerReader struct {
	dataChan     chan *protocol.Message
	disconnected chan struct{}
	assembler    *MessageAssembler
}

func NewPeerReader(connectionCount int, dataChan chan *protocol.Message, disconnected chan struct{}) *PeerReader {
	return &PeerReader{
		assembler:    NewMessageAssembler(dataChan, connectionCount),
		dataChan:     dataChan,
		disconnected: disconnected,
	}
}

func (r *PeerReader) Serve() {
	r.assembler.Serve()
}

func (r *PeerReader) Stop() {
	r.assembler.Stop()
}

func (r *PeerReader) AddConnection(conn *connection.Connection) {
	go func(c *connection.Connection) {
		reader := bufio.NewReader(c.GetConn())
		for {
			p, err := protocol.ReadPackage(reader)
			if err != nil {
				r.disconnected <- struct{}{}
				return
			}

			if p.Header.TotalPackageCount == 1 {
				var m protocol.Message
				r.dataChan <- m.FromPackages([]*protocol.Package{p})
			} else {
				r.assembler.AddPart(p)
			}
		}
	}(conn)
}

type MessageAssembler struct {
	pendings map[uint64][]*protocol.Package
	in       chan *protocol.Package
	out      chan *protocol.Message
}

func NewMessageAssembler(out chan *protocol.Message, concurrency int) *MessageAssembler {
	return &MessageAssembler{
		pendings: make(map[uint64][]*protocol.Package),
		in:       make(chan *protocol.Package, concurrency*2),
		out:      out,
	}
}

func (ma *MessageAssembler) Serve() {
	for p := range ma.in {
		ma.pendings[p.Header.ID] = append(ma.pendings[p.Header.ID], p)
		if uint32(len(ma.pendings[p.Header.ID])) == ma.pendings[p.Header.ID][0].Header.TotalPackageCount {
			var m protocol.Message
			m.FromPackages(ma.pendings[p.Header.ID])
			delete(ma.pendings, p.Header.ID)
			ma.out <- &m
		}
	}
}

func (ma *MessageAssembler) AddPart(p *protocol.Package) {
	ma.in <- p
}

func (ma *MessageAssembler) Stop() {
	close(ma.in)
}
