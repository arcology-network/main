/*
 *   Copyright (c) 2023 Arcology Network

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

package querycache

import (
	"math/big"

	ethcoretypes "github.com/arcology-network/evm/core/types"
)

type Block struct {
	Height    uint64
	Time      *big.Int
	TxNumbers int
	Hash      string
	GasUsed   uint64
}

type Transaction struct {
	TxHash   string
	Height   uint64
	Time     *big.Int
	Sender   string
	SendTo   string
	Amount   *big.Int
	Status   string
	GasLimit uint64
	GasPrice *big.Int
}

type CachedTx struct {
	Hash        string                // Transaction hash.
	Tx          []byte                // The encoded transaction.
	Height      uint64                // Block height.
	Index       uint64                // The index of the transaction in the block.
	ReceiptHash string                // The hash of the transaction receipt.
	Receipt     *ethcoretypes.Receipt // Transaction receipt.
	ExecRecipt  interface{}           // The execution receipt of the transaction.
}

type CachedBlock struct {
	Hash   string
	Height uint64
	Block  *ethcoretypes.Block
}

// func (this *QueryCache) BlockByHeight(blockHeight int) (interface{}, error) {
// 	return this.cache.Txn(false).First("block", "Height", blockHeight)
// }

// func (this *QueryCache) BlockByHash(hash string) (interface{}, error) {
// 	return this.cache.Txn(false).First("block", "Hash", hash)
// }

// func (this *QueryCache) TxByHash(blockHeight int) (interface{}, error) {
// 	return this.cache.Txn(false).First("block", "Height", blockHeight)
// }

// func (this *QueryCache) TxByBlockNumAndIdx(blockHeight, idx uint64) (interface{}, error) {
// 	return this.cache.Txn(false).First("block", "Height", idx)
// }
