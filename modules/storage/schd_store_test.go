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

package storage

import (
	"testing"

	evmCommon "github.com/ethereum/go-ethereum/common"
)

func TestWriteRead(t *testing.T) {
	store := NewSchdStore(1, "tester").(*SchdStore)
	store.Config(map[string]interface{}{
		"root": "./schdstore/",
	})

	store.writeToFile(&SchdState{
		Height:            1,
		NewContracts:      []evmCommon.Address{evmCommon.BytesToAddress([]byte{1}), evmCommon.BytesToAddress([]byte{2})},
		ConflictionLefts:  []evmCommon.Address{evmCommon.BytesToAddress([]byte{3}), evmCommon.BytesToAddress([]byte{4})},
		ConflictionRights: []evmCommon.Address{evmCommon.BytesToAddress([]byte{5}), evmCommon.BytesToAddress([]byte{6})},
	})
	store.writeToFile(&SchdState{
		Height:            2,
		NewContracts:      []evmCommon.Address{evmCommon.BytesToAddress([]byte{7}), evmCommon.BytesToAddress([]byte{8})},
		ConflictionLefts:  []evmCommon.Address{},
		ConflictionRights: []evmCommon.Address{},
	})
	store.writeToFile(&SchdState{
		Height:            3,
		NewContracts:      []evmCommon.Address{},
		ConflictionLefts:  []evmCommon.Address{evmCommon.BytesToAddress([]byte{9}), evmCommon.BytesToAddress([]byte{10})},
		ConflictionRights: []evmCommon.Address{evmCommon.BytesToAddress([]byte{11}), evmCommon.BytesToAddress([]byte{12})},
	})

	var states []SchdState
	store.readFromFile(&states)
	t.Log(states)
}
