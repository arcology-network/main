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
	"testing"

	"github.com/arcology-network/main/modules/p2p/conn/protocol"
	"github.com/arcology-network/streamer/actor"
)

func TestMessageToPackages(t *testing.T) {
	data, _ := (&actor.Message{
		Name: actor.MsgP2pRequest,
		Data: &P2pMessage{
			Sender: "node0",
			Message: &actor.Message{
				Name: actor.MsgSyncStatusRequest,
			},
		},
	}).Encode()
	t.Log(data)

	packages := protocol.Message{
		Type: protocol.MessageTypeClientBroadcast,
		Data: data,
	}.ToPackages()
	t.Log(packages)
}
