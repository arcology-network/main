//go:build NOCI
// +build NOCI

package connection_test

import (
	"testing"

	"github.com/HPISTechnologies/main/modules/p2p/conn/config"
	"github.com/HPISTechnologies/main/modules/p2p/conn/connection"
)

func TestConnectTimeout(t *testing.T) {
	cfg := &config.PeerConfig{
		Host: "localhost",
		Port: 11111,
	}

	_, err := connection.Connect("client", cfg)
	t.Log(err)
}

func TestNewConnection(t *testing.T) {
	cfg := &config.PeerConfig{
		ID:   "server",
		Host: "localhost",
		Port: 9292,
	}

	_, err := connection.Connect("client", cfg)
	if err != nil {
		t.Error(err)
	}
}

func TestHandshake(t *testing.T) {
	cfg := &config.PeerConfig{
		ID:   "server",
		Host: "localhost",
		Port: 9292,
	}

	conn, err := connection.Connect("client", cfg)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = conn.Handshake()
	if err != nil {
		t.Error(err)
		return
	}

	err = conn.Close()
	if err != nil {
		t.Error(err)
	}
}
