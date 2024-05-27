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
	"fmt"
	"math/big"
	"testing"

	"github.com/arcology-network/common-lib/tools"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

func TestExecutorRequestEncodingAndDeconing(t *testing.T) {
	from := ethCommon.BytesToAddress([]byte{0, 0, 0, 5, 6, 7, 8, 9})
	to := ethCommon.BytesToAddress([]byte{11, 12, 13})
	from0 := ethCommon.BytesToAddress([]byte{0, 0, 0, 1, 4, 5, 6, 7, 8})
	to0 := ethCommon.BytesToAddress([]byte{11, 12, 13, 14})

	ethMsg_serial_0 := core.NewMessage(from0, &to0, 1, big.NewInt(int64(1)), 100, big.NewInt(int64(8)), []byte{1, 2, 3}, nil, false)
	ethMsg_serial_1 := core.NewMessage(from, &to, 3, big.NewInt(int64(100)), 200, big.NewInt(int64(9)), []byte{4, 5, 6}, nil, false)
	fmt.Printf("ethMsg_serial_0=%v\n", ethMsg_serial_0)
	fmt.Printf("ethMsg_serial_1=%v\n", ethMsg_serial_1)
	hash1 := tools.RlpHash(ethMsg_serial_0)
	hash2 := tools.RlpHash(ethMsg_serial_1)
	fmt.Printf("hash1=%v\n", hash1)
	fmt.Printf("hash2=%v\n", hash2)
}
