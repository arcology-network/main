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
	"time"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type ReceiptCaches struct {
	caches      *DataCache
	db          *RawFile
	concurrency int
}

func NewReceiptCaches(path string, cache int, concurrency int) *ReceiptCaches {
	return &ReceiptCaches{
		caches:      NewDataCache(cache),
		db:          NewRawFiles(path),
		concurrency: concurrency,
	}
}
func (rc *ReceiptCaches) QueryReceipt(height uint64, idx int) *evmTypes.Receipt {
	data := rc.updateCache(height)
	if data == nil {
		return nil
	}
	return data[idx].(*evmTypes.Receipt)
}

func (rc *ReceiptCaches) QueryBlockReceipts(height uint64) []*evmTypes.Receipt {
	data := rc.updateCache(height)
	if data == nil {
		return nil
	}
	receipts := make([]*evmTypes.Receipt, len(data))
	for i := range receipts {
		receipts[i] = data[i].(*evmTypes.Receipt)
	}
	return receipts
}

func (rc *ReceiptCaches) updateCache(height uint64) []interface{} {
	retDatas := rc.caches.QueryBlock(height)
	if retDatas != nil {
		return retDatas
	}
	data, err := rc.db.Read(rc.db.GetFilename(height))
	if err != nil || data == nil {
		return nil
	}
	buffers := [][]byte(codec.Byteset{}.Decode(data).(codec.Byteset))
	receiptData := make([]interface{}, len(buffers))

	keys := make([]string, len(buffers))
	worker := func(start, end int, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			receiptobj := evmTypes.Receipt{}
			err = common.GobDecode(buffers[i], &receiptobj)
			if err != nil {
				continue
			}
			receiptData[i] = &receiptobj
			keys[i] = string(receiptobj.TxHash.Bytes())
		}
	}
	common.ParallelWorker(len(buffers), rc.concurrency, worker)
	rc.caches.Add(height, keys, receiptData)
	return receiptData
}

func (rc *ReceiptCaches) Save(height uint64, receipts []*evmTypes.Receipt) ([]string, []time.Duration) {
	tims := make([]time.Duration, 3)
	if len(receipts) == 0 {
		return []string{}, tims
	}
	t0 := time.Now()
	data := make([]interface{}, len(receipts))
	databyteset := make([][]byte, len(receipts))
	keys := make([]string, len(receipts))
	worker := func(start, end int, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			receiptRaw, err := common.GobEncode(receipts[i])
			if err != nil {
				continue
			}

			databyteset[i] = receiptRaw
			keys[i] = string(receipts[i].TxHash.Bytes())

			data[i] = receipts[i]

		}
	}

	common.ParallelWorker(len(receipts), rc.concurrency, worker)
	tims[0] = time.Since(t0)
	t0 = time.Now()
	rc.caches.Add(height, keys, data)
	tims[1] = time.Since(t0)
	t0 = time.Now()
	rc.db.Write(rc.db.GetFilename(height), codec.Byteset(databyteset).Encode())
	tims[2] = time.Since(t0)
	return keys, tims
}
