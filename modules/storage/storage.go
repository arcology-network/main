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
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

var (
	receiptRequest = promauto.NewSummary(prometheus.SummaryOpts{
		Name: "storage_receipt_request_process_seconds",
		Help: "The response time of the receipt request.",
	})

	storageSingleton actor.IWorkerEx
	initOnce         sync.Once
)

type Storage struct {
	actor.WorkerThread
	caches       *mstypes.LogCaches
	scanCache    *mstypes.ScanCache
	cacheSvcPort string
	lastHeight   uint64
	chainID      *big.Int

	params map[string]interface{}
}

// return a Subscriber struct
func NewStorage(concurrency int, groupid string) actor.IWorkerEx {
	initOnce.Do(func() {
		storageSingleton = &Storage{}
		storageSingleton.(*Storage).Set(concurrency, groupid)
	})
	return storageSingleton
}

func (s *Storage) Inputs() ([]string, bool) {
	return []string{
		// actor.MsgBlockCompleted,
		actor.MsgParentInfo,
		actor.MsgSelectedReceipts,
		actor.MsgPendingBlock,

		actor.MsgConflictInclusive,
	}, true
}

func (s *Storage) Outputs() map[string]int {
	return map[string]int{}
}

func (s *Storage) Config(params map[string]interface{}) {
	mstypes.CreateDB(params)
	s.caches = mstypes.NewLogCaches(int(params["log_cache_size"].(float64)))
	s.chainID = params["chain_id"].(*big.Int)
	s.scanCache = mstypes.NewScanCache(
		int(params["block_cache_size"].(float64)),
		int(params["tx_cache_size"].(float64)),
		s.chainID,
	)
	s.cacheSvcPort = params["cache_svc_port"].(string)

	s.params = params

}

func (s *Storage) OnStart() {
	var na int
	intf.Router.Call("statestore", "GetHeight", &na, &s.lastHeight)
	c := cors.AllowAll()
	go http.ListenAndServe(":"+s.cacheSvcPort, c.Handler(NewHandler(s.scanCache, s.params)))
}

func (*Storage) Stop() {}

func (s *Storage) OnMessageArrived(msgs []*actor.Message) error {
	//var statedatas *storage.UrlUpdate
	result := "success"
	height := uint64(0)
	var receipts []*evmTypes.Receipt
	var block *mtypes.MonacoBlock

	inclusive := &types.InclusiveList{}
	var na int

	for _, v := range msgs {
		switch v.Name {
		// case actor.MsgBlockCompleted:
		// 	result = v.Data.(string)
		case actor.MsgParentInfo:
			parentinfo := v.Data.(*mtypes.ParentInfo)
			isnil, err := s.IsNil(parentinfo, "parentinfo")
			if isnil {
				return err
			}
			intf.Router.Call("statestore", "Save", &State{
				Height:        v.Height,
				ParentHash:    parentinfo.ParentHash,
				ParentRoot:    parentinfo.ParentRoot,
				ExcessBlobGas: parentinfo.ExcessBlobGas,
				BlobGasUsed:   parentinfo.BlobGasUsed,
			}, &na)
		case actor.MsgSelectedReceipts:
			for _, item := range v.Data.([]interface{}) {
				receipts = append(receipts, item.(*evmTypes.Receipt))
			}
		case actor.MsgPendingBlock:
			block = v.Data.(*mtypes.MonacoBlock)
			height = v.Height

		case actor.MsgConflictInclusive:
			inclusive = v.Data.(*types.InclusiveList)
		}
	}

	if actor.MsgBlockCompleted_Success == result {
		savet := time.Now()
		s.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>> storage start gather info", zap.Uint64("blockNo", height), zap.Int("receiptsSize", len(receipts)))

		if block != nil && block.Height > 0 {
			t0 := time.Now()
			intf.Router.Call("blockstore", "Save", block, &na)
			s.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>> block save", zap.Duration("time", time.Since(t0)))
		}

		mapReceipts := make(map[evmCommon.Hash]*evmTypes.Receipt, len(receipts))
		for _, receipt := range receipts {
			mapReceipts[receipt.TxHash] = receipt
		}

		blockHash := block.Hash()

		s.scanCache.BlockReceived(block, blockHash, mapReceipts)

		t0 := time.Now()

		conflictTxs := map[evmCommon.Hash]int{}
		if inclusive != nil {
			for i, hash := range inclusive.HashList {
				if !inclusive.Successful[i] {
					conflictTxs[hash] = i
				}
			}
		}

		failed := 0
		keys := make([]string, len(receipts))
		worker := func(start, end int, idx int, args ...interface{}) {
			for i := start; i < end; i++ {
				txhash := receipts[i].TxHash

				if _, ok := conflictTxs[txhash]; ok {
					receipts[i].Status = 0
					receipts[i].GasUsed = 0
				}

				if receipts[i].Status == 0 {
					failed = failed + 1
				}

				receipts[i].BlockHash = evmCommon.BytesToHash(blockHash)
				receipts[i].BlockNumber = big.NewInt(int64(block.Height))
				receipts[i].TransactionIndex = uint(i)

				for k := range receipts[i].Logs {
					receipts[i].Logs[k].BlockHash = receipts[i].BlockHash
					receipts[i].Logs[k].TxHash = receipts[i].TxHash
					receipts[i].Logs[k].TxIndex = receipts[i].TransactionIndex
				}
				keys[i] = string(receipts[i].TxHash.Bytes())
			}
		}
		common.ParallelWorker(len(receipts), s.Concurrency, worker)

		if len(receipts) > 0 {
			intf.Router.Call("receiptstore", "Save", &SaveReceiptsRequest{
				Height:   height,
				Receipts: receipts,
			}, &na)
			intf.Router.Call("indexerstore", "Save", &SaveIndexRequest{
				Height: height,
				keys:   keys,
				IsSave: true,
			}, &na)
			intf.Router.Call("indexerstore", "SaveBlockHash", &SaveIndexBlockHashRequest{
				Height: height,
				Hash:   string(blockHash),
				IsSave: true,
			}, &na)
			s.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>> receipt save", zap.Int("total", len(receipts)), zap.Int("failed", failed), zap.Duration("time", time.Since(t0)))
			s.caches.Add(height, receipts)
		}

		s.AddLog(log.LogLevel_Info, "<<<<<<<<<<<<<<<<<<<<< storage gather info completed", zap.Duration("save time", time.Since(savet)), zap.Uint64("blockNo", height))
		s.lastHeight = height
	}

	return nil
}
