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
