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
	"reflect"
	"testing"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

func TestMetaBlock(t *testing.T) {
	ethHash := ethCommon.BytesToHash([]byte{9, 9, 9, 9, 9, 9})
	in := &MetaBlock{
		Txs:      [][]byte{{1, 2}, {3, 4}, {5, 6}, {9, 8}, {7, 6}, {5, 4}, {4, 6}, {2, 7}, {8, 0}, {1, 9}},
		Hashlist: []ethCommon.Hash{ethHash, ethHash, ethHash, ethHash, ethHash, ethHash, ethHash, ethHash, ethHash, ethHash},
	}
	buffer := in.Encode()

	out := new(MetaBlock)
	out.GobDecode(buffer)

	if !reflect.DeepEqual(in, out) {
		t.Error("Error")
	}
}
