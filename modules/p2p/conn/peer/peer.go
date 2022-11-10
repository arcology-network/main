package peer

import (
	"net"

	"github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/arcology-network/main/modules/p2p/conn/connection"
	"github.com/arcology-network/main/modules/p2p/conn/protocol"
)

type Peer struct {
	id            string
	cfg           *config.PeerConfig
	connections   []*connection.Connection
	sendChan      chan *protocol.Message
	recvChan      chan *protocol.Message
	quit          chan struct{}
	reader        *PeerReader
	writer        *PeerWriter
	onMsgReceived func(string, *protocol.Message)
	ready         bool
}

func NewPeer(id string, cfg *config.PeerConfig, onMsgReceived func(string, *protocol.Message)) *Peer {
	p := &Peer{
		id:            id,
		cfg:           cfg,
		connections:   make([]*connection.Connection, 0, cfg.ConnectionCount),
		sendChan:      make(chan *protocol.Message, cfg.ConnectionCount*2),
		recvChan:      make(chan *protocol.Message, cfg.ConnectionCount*2),
		quit:          make(chan struct{}, cfg.ConnectionCount),
		onMsgReceived: onMsgReceived,
	}
	p.reader = NewPeerReader(int(p.cfg.ConnectionCount), p.recvChan, p.quit)
	go p.reader.Serve()
	p.writer = NewPeerWriter(int(p.cfg.ConnectionCount), p.sendChan, p.quit)
	go p.writer.Serve()

	return p
}

func (p *Peer) IsReady() bool {
	return p.ready
}

func (p *Peer) GetReady() {
	p.ready = true
}

func (p *Peer) AddConnection(conn net.Conn) {
	c, _ := connection.NewConnection(p.id, p.cfg, conn)
	p.connections = append(p.connections, c)
	p.reader.AddConnection(c)
	p.writer.AddConnection(c)
	// FIXME
	if len(p.connections) == int(p.cfg.ConnectionCount) {
		p.ready = true
	}
}

func (p *Peer) Serve(onClose func()) error {
	go func() {
		for m := range p.recvChan {
			p.onMsgReceived(p.id, m)
		}
	}()

	<-p.quit
	for _, c := range p.connections {
		c.Close()
	}
	// Wait for all the routines to exit.
	for i := 1; i < int(p.cfg.ConnectionCount); i++ {
		<-p.quit
	}
	close(p.recvChan)
	close(p.sendChan)
	close(p.quit)
	p.reader.Stop()
	p.writer.Stop()
	onClose()
	return nil
}

func (p *Peer) Connect() ([]*connection.Connection, error) {
	conn, err := connection.Connect(p.id, p.cfg)
	if err != nil {
		return nil, err
	}

	cfg, err := conn.Handshake()
	if err != nil {
		conn.Close()
		return nil, err
	}

	p.cfg.ConnectionCount = cfg.ConnectionCount
	p.connections = make([]*connection.Connection, 0, p.cfg.ConnectionCount)
	p.connections = append(p.connections, conn)

	for i := 1; i < int(p.cfg.ConnectionCount); i++ {
		conn, err := connection.Connect(p.id, p.cfg)
		if err != nil {
			p.Disconnect()
			return nil, err
		}

		_, err = conn.Handshake()
		if err != nil {
			conn.Close()
			p.Disconnect()
			return nil, err
		}
		p.connections = append(p.connections, conn)
	}
	return p.connections, nil
}

func (p *Peer) Disconnect() {
	for _, conn := range p.connections {
		conn.Close()
	}
}

func (p *Peer) Send(m *protocol.Message, pushIn bool) {
	if pushIn {
		p.writer.PushIn(m)
	} else {
		p.sendChan <- m
	}
}
