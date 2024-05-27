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

package p2p

import (
	"encoding/gob"

	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

func init() {
	actor.Factory.Register("p2p.gateway", NewP2pGateway)
	actor.Factory.Register("p2p.conn", NewP2pConn)
	actor.Factory.Register("p2p.client", NewP2pClient)

	intf.Factory.Register("p2p.conn", func(concurrency int, groupId string) interface{} {
		return NewP2pConn(concurrency, groupId)
	})

	gob.Register(&P2pMessage{})
}
