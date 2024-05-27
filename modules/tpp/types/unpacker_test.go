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
	"time"

	"github.com/arcology-network/common-lib/tools"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	evmRlp "github.com/ethereum/go-ethereum/rlp"
)

func TestRlp(t *testing.T) {
	txnums := 15000
	txs := make([]*evmTypes.Transaction, 0, txnums)

	for i := range txs {
		txs[i] = evmTypes.NewTransaction(
			0,
			evmCommon.BigToAddress(big.NewInt(int64(i))),
			big.NewInt(int64(i)),
			uint64(i),
			big.NewInt(int64(i)),
			[]byte{byte(i % 255), byte((i + 1) % 255), byte((i + 2) % 255)},
		)
	}
	txsRawData := make([][]byte, 0, txnums)
	startTime := time.Now()
	for i := range txs {
		data, err := evmRlp.EncodeToBytes(txs[i])
		if err != nil {
			fmt.Printf("eocode txs[%d] err: %v!\n", i, err)
			return
		}
		txsRawData[i] = data
	}
	fmt.Printf("encode txsnums=%v transactions time=%v ms\n", txnums, time.Now().Sub(startTime))
	startTime1 := time.Now()
	for i := range txsRawData {
		otx := new(evmTypes.Transaction)
		if err := evmRlp.DecodeBytes(txsRawData[i], otx); err != nil {
			fmt.Printf("deocode txs[%d] err: %v!\n", i, err)
			return
		}
		tools.RlpHash(otx)

	}
	fmt.Printf("dencode txsnums=%v transactions time=%v ms\n", txnums, time.Now().Sub(startTime1))
}
