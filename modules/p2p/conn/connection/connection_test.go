//go:build NOCI
// +build NOCI

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

package connection_test

import (
	"testing"

	"github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/arcology-network/main/modules/p2p/conn/connection"
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
