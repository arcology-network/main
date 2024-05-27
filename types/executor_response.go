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
	codec "github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/types"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

type ExecuteResponse struct {
	Hash    ethCommon.Hash
	Status  uint64
	GasUsed uint64
}

type ExecutorResponses struct {
	HashList    []ethCommon.Hash
	StatusList  []uint64
	GasUsedList []uint64

	ContractAddresses []ethCommon.Address

	CallResults [][]byte
}

func (er *ExecutorResponses) GobEncode() ([]byte, error) {
	data := [][]byte{
		types.Hashes(er.HashList).Encode(),
		codec.Uint64s(er.StatusList).Encode(),
		codec.Uint64s(er.GasUsedList).Encode(),

		types.Addresses(er.ContractAddresses).Encode(),

		codec.Byteset(er.CallResults).Encode(),
	}
	return codec.Byteset(data).Encode(), nil
}
func (er *ExecutorResponses) GobDecode(data []byte) error {
	fields := codec.Byteset{}.Decode(data).(codec.Byteset)
	er.HashList = types.Hashes(er.HashList).Decode(fields[0])
	er.StatusList = []uint64(codec.Uint64s(er.StatusList).Decode(fields[1]).(codec.Uint64s))
	er.GasUsedList = []uint64(codec.Uint64s(er.GasUsedList).Decode(fields[2]).(codec.Uint64s))

	er.ContractAddresses = types.Addresses(er.ContractAddresses).Decode(fields[3])

	er.CallResults = [][]byte(codec.Byteset{}.Decode(fields[4]).(codec.Byteset))
	return nil
}
