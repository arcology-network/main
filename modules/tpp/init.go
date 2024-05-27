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

package tpp

import (
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

func init() {
	actor.Factory.Register("tpp_rpc", NewRpcReceiver)
	actor.Factory.Register("tx_receiver", NewTxReceiver)
	actor.Factory.Register("tx_unsigner", NewTxUnsigner)

	intf.Factory.Register("tpp_rpc", func(concurrency int, groupId string) interface{} {
		return NewRpcReceiver(concurrency, groupId)
	})
}
