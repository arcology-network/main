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

package backend

import (
	"fmt"
	"math/big"
	"testing"

	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

func TestMsgHash(t *testing.T) {
	msg := core.NewMessage(
		evmCommon.BytesToAddress([]byte{1}),
		nil,
		1,
		new(big.Int).SetUint64(1000),
		2000,
		new(big.Int).SetUint64(3000),
		[]byte{4},
		nil,
		false,
	)
	hash, _ := msgHash(&msg)
	t.Log(hash)
}
func TestID(t *testing.T) {
	fmt.Printf("%v", NewID())
}
