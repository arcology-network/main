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

package types

import (
	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/types"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

type MetaBlock struct {
	Txs      [][]byte
	Hashlist []ethCommon.Hash
}

func (this MetaBlock) HeaderSize() uint64 {
	return uint64(3 * codec.UINT64_LEN)
}

func (this MetaBlock) Size() uint64 {
	total := 0
	for i := 0; i < len(this.Txs); i++ {
		total += len(this.Txs[i])
	}
	return uint64(
		this.HeaderSize() +
			uint64(codec.UINT64_LEN*(len(this.Txs)+1)) + uint64(total) +
			uint64(len(this.Hashlist)*codec.HASH32_LEN))
}

func (this MetaBlock) Encode() []byte {
	buffer := make([]byte, this.Size())
	this.EncodeToBuffer(buffer)
	return buffer
}

func (this MetaBlock) FillHeader(buffer []byte) {
	codec.Uint64(2).EncodeToBuffer(buffer[codec.UINT64_LEN*0:])
	codec.Uint64(0).EncodeToBuffer(buffer[codec.UINT64_LEN*1:])
	codec.Uint64(codec.Byteset(this.Txs).Size()).EncodeToBuffer(buffer[codec.UINT64_LEN*2:])
}

func (this MetaBlock) EncodeToBuffer(buffer []byte) {
	this.FillHeader(buffer)
	headerLen := this.HeaderSize()

	offset := uint64(0)
	codec.Byteset(this.Txs).EncodeToBuffer(buffer[headerLen+offset:])
	offset += codec.Byteset(this.Txs).Size()

	for i := 0; i < len(this.Hashlist); i++ {
		codec.Bytes32(this.Hashlist[i]).EncodeToBuffer(buffer[headerLen+offset:])
		offset += uint64(ethCommon.HashLength)
	}
}

func (this MetaBlock) GobEncode() ([]byte, error) {
	return this.Encode(), nil
}

func (this *MetaBlock) GobDecode(buffer []byte) error {
	fields := codec.Byteset{}.Decode(buffer).(codec.Byteset)
	this.Txs = codec.Byteset{}.Decode(fields[0]).(codec.Byteset)
	arrs := types.Hashes([]ethCommon.Hash{}).Decode(fields[1])
	this.Hashlist = arrs
	return nil
}
