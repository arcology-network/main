//go:build NOCI
// +build NOCI

package peer

import (
	"fmt"
	"testing"
	"time"

	"github.com/HPISTechnologies/main/modules/p2p/conn/config"
	"github.com/HPISTechnologies/main/modules/p2p/conn/protocol"
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
