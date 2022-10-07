package connection

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/HPISTechnologies/main/modules/p2p/conn/config"
	"github.com/HPISTechnologies/main/modules/p2p/conn/protocol"
)

type Connection struct {
	id     string
	conn   net.Conn
	reader *bufio.Reader
	cfg    *config.PeerConfig
}

func Connect(id string, cfg *config.PeerConfig) (*Connection, error) {
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		return nil, err
	}

	return &Connection{
		id:     id,
		conn:   conn,
		reader: bufio.NewReader(conn),
		cfg:    cfg,
	}, nil
}

func NewConnection(id string, cfg *config.PeerConfig, conn net.Conn) (*Connection, error) {
	return &Connection{
		id:     id,
		conn:   conn,
		reader: bufio.NewReader(conn),
		cfg:    cfg,
	}, nil
}

func (c *Connection) Close() error {
	return c.conn.Close()
}

func (c *Connection) Handshake() (*config.PeerConfig, error) {
	err := protocol.WriteMessage(c.conn, protocol.MsgHandshake{ID: c.id, ConnectionCount: c.cfg.ConnectionCount}.ToMessage())
	if err != nil {
		return c.cfg, err
	}

	m, err := protocol.ReadMessage(c.reader)
	if err != nil {
		return c.cfg, err
	}

	// Validation.
	if m.Type != protocol.MessageTypeHandshake {
		return c.cfg, errors.New("peer validation failed")
	}
	var msg protocol.MsgHandshake
	msg.FromMessage(m)
	c.cfg.ID = msg.ID
	c.cfg.ConnectionCount = msg.ConnectionCount

	return c.cfg, nil
}

func (c *Connection) Routing(host string, port int) (*protocol.MsgRouting, error) {
	err := protocol.WriteMessage(c.conn, protocol.MsgRouting{Host: host, Port: port}.ToMessage())
	if err != nil {
		return nil, err
	}

	m, err := protocol.ReadMessage(c.reader)
	if err != nil {
		return nil, err
	}

	if m.Type != protocol.MessageTypeRouting {
		return nil, errors.New(fmt.Sprintf("unexpected message type: %v", m.Type))
	}
	var msg protocol.MsgRouting
	return msg.FromMessage(m), nil
}

func (c *Connection) GetConn() net.Conn {
	return c.conn
}
