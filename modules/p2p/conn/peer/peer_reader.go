package peer

import (
	"bufio"

	"github.com/HPISTechnologies/main/modules/p2p/conn/connection"
	"github.com/HPISTechnologies/main/modules/p2p/conn/protocol"
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
			// fmt.Printf("new message, id = %v, len = %v\n", m.ID, len(m.Data))
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
