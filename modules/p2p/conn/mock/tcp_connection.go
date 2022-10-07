package mock

import (
	"net"
	"time"
)

type TCPConnection struct {
	queue chan []byte
	buf   []byte
}

func NewTCPConnection() *TCPConnection {
	return &TCPConnection{
		queue: make(chan []byte, 1024),
	}
}

func (c *TCPConnection) Read(b []byte) (int, error) {
	next := <-c.queue
	copy(b, next)
	return len(next), nil
	// for len(c.buf) < len(b) {
	// 	next := <-c.queue
	// 	c.buf = append(c.buf, next...)
	// }

	// copy(b, c.buf)
	// c.buf = c.buf[len(b):]
	// return len(b), nil
}

func (c *TCPConnection) Write(b []byte) (int, error) {
	c.queue <- b
	return len(b), nil
}

func (c *TCPConnection) Close() error {
	return nil
}

func (c *TCPConnection) LocalAddr() net.Addr {
	return nil
}

func (c *TCPConnection) RemoteAddr() net.Addr {
	return nil
}

func (c *TCPConnection) SetDeadline(t time.Time) error {
	return nil
}

func (c *TCPConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *TCPConnection) SetWriteDeadline(t time.Time) error {
	return nil
}
