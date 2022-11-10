package peer

import (
	"fmt"
	"time"

	"github.com/arcology-network/main/modules/p2p/conn/connection"
	"github.com/arcology-network/main/modules/p2p/conn/protocol"
)

type PeerWriter struct {
	pushInChan   chan *protocol.Message
	dataChan     chan *protocol.Message
	disconnected chan struct{}
	dispatchChan chan *protocol.Package
}

func NewPeerWriter(connectionCount int, dataChan chan *protocol.Message, disconnected chan struct{}) *PeerWriter {
	return &PeerWriter{
		pushInChan:   make(chan *protocol.Message, connectionCount),
		dataChan:     dataChan,
		disconnected: disconnected,
		dispatchChan: make(chan *protocol.Package, connectionCount*2),
	}
}

func (w *PeerWriter) AddConnection(conn *connection.Connection) {
	go func(c *connection.Connection) {
		for {
			p := <-w.dispatchChan
			if p == nil {
				return
			}
			err := protocol.WritePackage(c.GetConn(), p)
			if err != nil {
				w.disconnected <- struct{}{}
				return
			}
		}
	}(conn)
}

func (w *PeerWriter) Serve() {
	id := uint64(0)
	for {
		var msg *protocol.Message
		select {
		case msg = <-w.pushInChan:
		default:
			select {
			case msg = <-w.pushInChan:
			case msg = <-w.dataChan:
			case <-time.After(30 * time.Millisecond):
				continue
			}
		}

		if msg == nil {
			continue
		}
		msg.ID = id
		packages := msg.ToPackages()
		fmt.Printf("[PeerWriter.Serve] %d packages to write\n", len(packages))
		for _, p := range packages {
			w.dispatchChan <- p
		}
		id++
	}
}

func (w *PeerWriter) PushIn(msg *protocol.Message) {
	// fmt.Printf("[PeerWriter.PushIn] msg = %v\n", msg)
	w.pushInChan <- msg
}

func (w *PeerWriter) Stop() {
	close(w.dispatchChan)
}
