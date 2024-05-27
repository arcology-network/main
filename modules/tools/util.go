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

package tools

import (
	"github.com/arcology-network/common-lib/exp/slice"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func CalculateHash(hashes []*evmCommon.Hash) evmCommon.Hash {
	if len(hashes) == 0 {
		return evmCommon.Hash{}
	}
	return evmCommon.BytesToHash(crypto.Keccak256(slice.Concate(hashes, func(v *evmCommon.Hash) []byte { return (*v)[:] })))
}
