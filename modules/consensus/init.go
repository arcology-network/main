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

package consensus

import (
	"crypto/sha256"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/consensus-engine/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

func init() {
	types.QuickHash = CalculateHash

	actor.Factory.Register("consensus", NewConsensus)
	intf.Factory.Register("consensus", func(concurrency int, groupId string) interface{} {
		return NewConsensus(concurrency, groupId)
	})
}

func CalculateHash(txs types.Txs) ([]byte, error) {
	data := make([][]byte, len(txs))
	for i := range txs {
		data[i] = txs[i]
	}
	hash := sha256.Sum256(codec.Byteset(data).Flatten())
	return hash[:], nil
}
