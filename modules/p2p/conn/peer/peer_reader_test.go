package peer

import (
	"bytes"
	"testing"

	"github.com/HPISTechnologies/main/modules/p2p/conn/config"
	"github.com/HPISTechnologies/main/modules/p2p/conn/connection"
	"github.com/HPISTechnologies/main/modules/p2p/conn/mock"
	"github.com/HPISTechnologies/main/modules/p2p/conn/protocol"
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
