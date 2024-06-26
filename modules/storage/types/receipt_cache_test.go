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
	"os"
	"reflect"
	"testing"

	"github.com/arcology-network/common-lib/storage/filedb"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

func newReceipt(height uint64, idx, idxInBlock int) *evmTypes.Receipt {
	receipt := evmTypes.Receipt{}
	receipt.PostState = []byte{byte(idx + 1), 2, 3, 4}
	receipt.Status = 1
	receipt.CumulativeGasUsed = 100
	receipt.Bloom = evmTypes.BytesToBloom([]byte{2, byte(idx + 3), 4, 5, 6})
	receipt.TxHash = evmCommon.HexToHash(hashes[idx])
	receipt.ContractAddress = evmCommon.BytesToAddress([]byte{1, 2, 3, 4, 5, byte(idx + 6)})
	receipt.GasUsed = 2100
	receipt.Type = 1
	receipt.BlockHash = evmCommon.HexToHash(hashes[idx])
	receipt.BlockNumber = big.NewInt(int64(height))
	receipt.TransactionIndex = uint(idxInBlock)
	return &receipt
}

func TestReceiptCache(t *testing.T) {
	filedb, err := filedb.NewFileDB("index", 16, 2)
	if err != nil {
		panic("create filedb err!:" + err.Error())
	}
	cacheSize := 2
	indexer := NewIndexer(filedb, cacheSize)
	cache := NewReceiptCaches("receiptfiles", cacheSize, 8)

	receipts1 := make([]*evmTypes.Receipt, 2)

	for i := range receipts1 {
		receipts1[i] = newReceipt(1, 0+i, i)
	}
	keys, _ := cache.Save(1, receipts1)
	indexer.Add(1, keys, true)
	indexer.AddBlockHashHeight(1, string(receipts1[0].BlockHash.Bytes()), true)

	position := indexer.QueryPosition(string(receipts1[1].TxHash.Bytes()))
	if position == nil {
		t.Error("cache index save get Error")
		return
	}
	queryResult := cache.QueryReceipt(position.Height, position.IdxInBlock)
	if !reflect.DeepEqual(*queryResult, *receipts1[1]) {
		t.Error("cache save get Error")
		return
	}

	receipts2 := make([]*evmTypes.Receipt, 2)

	for i := range receipts2 {
		receipts2[i] = newReceipt(2, 2+i, i)
	}
	keys, _ = cache.Save(2, receipts2)
	indexer.Add(2, keys, true)
	indexer.AddBlockHashHeight(2, string(receipts2[0].BlockHash.Bytes()), true)

	position = indexer.QueryPosition(string(receipts2[1].TxHash.Bytes()))
	if position == nil {
		t.Error("cache index save get Error")
		return
	}
	queryResult = cache.QueryReceipt(position.Height, position.IdxInBlock)

	if !reflect.DeepEqual(*queryResult, *receipts2[1]) {
		t.Error("cache save get Error")
		return
	}

	receipts3 := make([]*evmTypes.Receipt, 2)

	for i := range receipts3 {
		receipts3[i] = newReceipt(3, 4+i, i)
	}
	keys, _ = cache.Save(3, receipts3)
	indexer.Add(3, keys, true)
	indexer.AddBlockHashHeight(3, string(receipts3[0].BlockHash.Bytes()), true)

	heights := []uint64{2, 3}
	if !reflect.DeepEqual(heights, indexer.Caches.DataHeights) {
		t.Error("cache remove Error")
		return
	}

	position = indexer.QueryPosition(string(receipts1[0].TxHash.Bytes()))
	if position == nil {
		t.Error("cache index save get Error")
		return
	}
	queryResult = cache.QueryReceipt(position.Height, position.IdxInBlock)

	if !reflect.DeepEqual(*queryResult, *receipts1[0]) {
		t.Error("cache save get Error")
		return
	}
	heights = []uint64{3, 1}
	if !reflect.DeepEqual(heights, indexer.Caches.DataHeights) {
		t.Error("cache remove Error")
		return
	}

	queryReceipt := cache.QueryReceipt(3, 0)
	if !reflect.DeepEqual(*queryReceipt, *receipts3[0]) {
		t.Error("query receipt Error")
		return
	}

	height := indexer.QueryBlockHashHeight(string(receipts3[0].BlockHash.Bytes()))
	if !reflect.DeepEqual(height, uint64(3)) {
		t.Error("query height by block hash Error")
		return
	}

	position = indexer.QueryPosition(string(receipts3[0].BlockHash.Bytes()))
	if !reflect.DeepEqual(position.Height, uint64(3)) {
		t.Error("query position height by block hash Error")
		return
	}
	if !reflect.DeepEqual(position.IdxInBlock, 0) {
		t.Error("query position idx by block hash Error")
		return
	}

	hs := indexer.GetBlockHashesByHeightFromCache(2)
	txhash2 := make([]string, 2)
	for i := range txhash2 {
		txhash2[i] = fmt.Sprintf("%x", receipts2[i].TxHash.Bytes())
	}
	if !reflect.DeepEqual(hs, txhash2) {
		t.Error("indexer.GetBlockHashesByHeightFromCache Error")
		return
	}
	os.RemoveAll("index")
	os.RemoveAll("receiptfiles")
}
