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
	"log"
	"math/big"
	"time"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	queryplan "github.com/arcology-network/main/modules/storage/query_plan"
	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evm "github.com/ethereum/go-ethereum"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/arcology-network/main/modules/storage/query"
)

var (
	receiptRequest = promauto.NewSummary(prometheus.SummaryOpts{
		Name: "storage_receipt_request_process_seconds",
		Help: "The response time of the receipt request.",
	})
)

type Storage struct {
	caches    *mstypes.LogCaches
	scanCache *mstypes.ScanCache
	// cacheSvcPort string
	lastHeight uint64
	chainID    *big.Int

	// params map[string]interface{}

	//save context
	receipts   []*evmTypes.Receipt
	block      *mtypes.MonacoBlock
	inclusive  *types.InclusiveList
	parentinfo *mtypes.ParentInfo
	height     uint64
	startTime  time.Time
	keys       []string
	blockHash  []byte
	failed     int

	//-----------------------
	dispatcher *query.Dispatcher
	scheduler  *query.MockScheduler
	observer   *query.MockObserver
}

// return a Subscriber struct
func NewStorage() actor.Business {
	s := &Storage{
		dispatcher: query.NewDispatcher(),
		scheduler:  &query.MockScheduler{},
		observer:   &query.MockObserver{},
	}
	RegisterQuery(s)
	return s
}

func (s *Storage) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgParentInfo,
		scommon.MsgSelectedReceipts,
		scommon.MsgPendingBlock,
		scommon.MsgConflictInclusive,
	}, true
}
func (s *Storage) RpcConfig() (string, int) {
	return "storage", 20
}

func (s *Storage) Outputs() map[string]int {
	return map[string]int{}
}

func (s *Storage) PrimaryMsg() string {
	return scommon.MsgPendingBlock
}

func (s *Storage) Config(params map[string]interface{}) {
	mstypes.CreateDB(params)
	s.caches = mstypes.NewLogCaches(params["log_cache_size"].(int))
	s.chainID = params["chain_id"].(*big.Int)
	s.scanCache = mstypes.NewScanCache(
		params["block_cache_size"].(int),
		params["tx_cache_size"].(int),
		s.chainID,
	)
	// s.cacheSvcPort = params["cache_svc_port"].(string)
	// s.params = params

	// c := cors.AllowAll()
	// go http.ListenAndServe(":"+s.cacheSvcPort, c.Handler(NewHandler(s.scanCache, s.params)))
}

func (s *Storage) InitHeight(ctx *actor.ActionContext) error {
	s.lastHeight = ctx.RPC.Request.(uint64)
	ctx.ExecCtx.SendRpcResponse("", "")
	return nil
}

func (s *Storage) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgParentInfo, s.startSave)
	reg.Register(scommon.MsgSelectedReceipts, s.startSave)
	reg.Register(scommon.MsgPendingBlock, s.startSave)
	reg.Register(scommon.MsgConflictInclusive, s.startSave)
	// reg.Register("Query", s.Query)
	reg.Register("saveBlock", s.saveBlock)
	reg.Register("saveReceipt", s.saveReceipt)
	reg.Register("indexerSave", s.indexerSave)
	reg.Register("saveCache", s.saveCache)

	reg.Register("Query", s.Query)
	reg.Register("QueryContinuationAction", s.QueryContinuationAction)
	reg.Register("InitHeight", s.InitHeight)
}

func (s *Storage) startSave(ctx *actor.ActionContext) error {
	for _, v := range ctx.Messages {
		switch v.Name {
		case scommon.MsgParentInfo:
			s.parentinfo = v.Data.(*mtypes.ParentInfo)
		case scommon.MsgSelectedReceipts:
			s.receipts = v.Data.([]*evmTypes.Receipt)
		case scommon.MsgPendingBlock:
			s.block = v.Data.(*mtypes.MonacoBlock)
			s.height = v.Height
		case scommon.MsgConflictInclusive:
			s.inclusive = v.Data.(*types.InclusiveList)
		}
	}

	ctx.ExecCtx.LogDebug("storage start gather info", logger.F("blockNo", s.height), logger.F("receiptsSize", len(s.receipts)))

	ctx.ExecCtx.InvokeRPC("statestore", "Save", &State{
		Height:        s.height,
		ParentHash:    s.parentinfo.ParentHash,
		ParentRoot:    s.parentinfo.ParentRoot,
		ExcessBlobGas: s.parentinfo.ExcessBlobGas,
		BlobGasUsed:   s.parentinfo.BlobGasUsed,
	}, "saveBlock")

	return nil
}
func (s *Storage) saveBlock(ctx *actor.ActionContext) error {
	s.startTime = time.Now()
	ctx.ExecCtx.InvokeRPC("blockstore", "Save", s.block, "saveReceipt")
	return nil
}
func (s *Storage) saveReceipt(ctx *actor.ActionContext) error {
	ctx.ExecCtx.LogDebug("block save", logger.F("time", time.Since(s.startTime)))

	mapReceipts := make(map[evmCommon.Hash]*evmTypes.Receipt, len(s.receipts))
	for _, receipt := range s.receipts {
		mapReceipts[receipt.TxHash] = receipt
	}

	blockHash := s.block.Hash()
	s.scanCache.BlockReceived(s.block, blockHash, mapReceipts)

	s.startTime = time.Now()

	conflictTxs := map[evmCommon.Hash]int{}
	if s.inclusive != nil {
		for i, hash := range s.inclusive.HashList {
			if !s.inclusive.Successful[i] {
				conflictTxs[hash] = i
			}
		}
	}

	failed := 0
	keys := make([]string, len(s.receipts))
	worker := func(start, end int, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			txhash := s.receipts[i].TxHash

			if _, ok := conflictTxs[txhash]; ok {
				s.receipts[i].Status = 0
				s.receipts[i].GasUsed = 0
			}

			if s.receipts[i].Status == 0 {
				failed = failed + 1
			}

			s.receipts[i].BlockHash = evmCommon.BytesToHash(blockHash)
			s.receipts[i].BlockNumber = big.NewInt(int64(s.block.Height))
			s.receipts[i].TransactionIndex = uint(i)
			if s.receipts[i].Status == 0 {
				s.receipts[i].Logs = []*evmTypes.Log{}
			} else {
				for k := range s.receipts[i].Logs {
					s.receipts[i].Logs[k].BlockHash = s.receipts[i].BlockHash
					s.receipts[i].Logs[k].TxHash = s.receipts[i].TxHash
					s.receipts[i].Logs[k].TxIndex = s.receipts[i].TransactionIndex
				}
			}
			keys[i] = string(s.receipts[i].TxHash.Bytes())
		}
	}
	common.ParallelWorker(len(s.receipts), ctx.ExecCtx.Concurrency(), worker)

	s.keys = keys
	s.blockHash = blockHash
	s.failed = failed

	ctx.ExecCtx.InvokeRPC("receiptstore", "Save", &SaveReceiptsRequest{
		Height:   s.height,
		Receipts: s.receipts,
	}, "indexerSave")
	return nil
}
func (s *Storage) indexerSave(ctx *actor.ActionContext) error {
	ctx.ExecCtx.InvokeRPC("indexerstore", "Save", &SaveIndexRequest{
		Height: s.height,
		Keys:   s.keys,
		Hash:   string(s.blockHash),
		IsSave: true,
	}, "saveCache")
	return nil
}
func (s *Storage) saveCache(ctx *actor.ActionContext) error {
	ctx.ExecCtx.LogDebug("receipt save", logger.F("total", len(s.receipts)), logger.F("failed", s.failed), logger.F("time", time.Since(s.startTime)))
	s.caches.Add(s.height, s.receipts)
	s.lastHeight = s.height
	return nil
}

//-----------------------for query---------

func (a *Storage) Query(ctx *actor.ActionContext) error {
	req := ctx.RPC.Request.(*mtypes.QueryRequest)

	ctx.ExecCtx.LogDebug(
		"Query plan received request",
		logger.F("QueryType", req.QueryType),
	)

	plan := a.dispatcher.GetQueryPlan(req.QueryType)
	if plan == nil {
		ctx.ExecCtx.SendRpcResponse(query.ErrPlanNotFound.Error(), nil)
		return nil
	}

	a.dispatcher.Query(
		req.QueryType,
		req.Data,
		ctx.ExecCtx,
		a.observer,
		a.scheduler,
		func(resp interface{}, err error) {

			// ctx.ExecCtx.EndCasecade()
			if err != nil {
				ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
				return
			}
			log.Printf("[CONT] action ereturn -- resp:%v", resp)
			ctx.ExecCtx.SendRpcResponse("", &mtypes.QueryResult{
				Data: resp,
			})
		},
	)

	return nil
}

func (a *Storage) QueryContinuationAction(ctx *actor.ActionContext) error {
	if len(ctx.Messages) == 0 {
		ctx.ExecCtx.LogErr("no messages")
		return nil
	}

	msg := ctx.Messages[0]

	log.Printf("[CONT] action enter -- contId:%s", msg.ContId)

	cont, ok := query.TakeContinuationById(msg.ContId)
	log.Printf("[CONT] take -- contId:%s ok:%v", msg.ContId, ok)

	if !ok {
		ctx.ExecCtx.LogErr("continuation not found", logger.F("contId", msg.ContId))
		return nil
	}

	cont(ctx.RPC.Request, nil)
	log.Printf("[CONT] take -- contId:%s cont END ", msg.ContId)
	return nil
}

// func (a *Storage) QueryContinuationAction(ctx *actor.ActionContext) error {
// 	if len(ctx.Messages) == 0 {
// 		ctx.ExecCtx.LogErr("no messages")
// 		return nil
// 	}
// 	msg := ctx.Messages[0]

// 	log.Printf("[CONT] action enter -- contId:%s messages_len:%s", msg.ContId, fmt.Sprintf("%v", len(ctx.Messages)))

// ctx.ExecCtx.LogDebug(
// 	"[QueryContinuationAction] message",
// 	logger.F("next_step", msg.NextStep),
// 	logger.F("payload", msg.Data),
// )
// ctx.ExecCtx.LogDebug(fmt.Sprintf("req:%v", ctx.RPC.Request))
// qctxAny := ctx.ExecCtx.GetLocal("query_ctx")
// if qctxAny == nil {
// 	ctx.ExecCtx.LogErr("query_ctx not found")
// 	return nil
// }
// qctx := qctxAny.(*query.QueryContext)

// cont, ok := qctx.Root.TakeContinuation(msg.ContId)
// 	cont, ok := query.TakeContinuationById(msg.ContId)

// 	log.Printf("[CONT] take -- contId:%s ok:%s", msg.ContId, fmt.Sprintf("%v", ok))

// 	if !ok {
// 		ctx.ExecCtx.LogErr("continuation not found", logger.F("contId", msg.ContId))
// 		return nil
// 	}
// 	ctx.ExecCtx.LogDebug("continuation hit")
// 	cont(ctx.RPC.Request, nil)
// 	return nil
// }

func RegisterQuery(s *Storage) {
	s.dispatcher.Register(mtypes.QueryType_RawBlock, &queryplan.GetRawBlockQueryPlan{})
	//--------------------------------
	s.dispatcher.Register(mtypes.QueryType_BlockNumber,
		&queryplan.InlineQueryPlan{
			Fn: func(ctx *query.QueryContext) (interface{}, error) {
				return s.lastHeight, nil
			},
		},
	)

	s.dispatcher.Register(mtypes.QueryType_TransactionCount, &queryplan.TransactionCountQueryPlan{})

	s.dispatcher.Register(mtypes.QueryType_Code, &queryplan.CodeQueryPlan{})

	s.dispatcher.Register(mtypes.QueryType_Balance_Eth, &queryplan.BalanceEthQueryPlan{})

	s.dispatcher.Register(mtypes.QueryType_Storage, &queryplan.StorageQueryPlan{})

	//-------------

	s.dispatcher.Register(mtypes.QueryType_TestBlockByHeight, &queryplan.TestBlockByHeightQueryPlan{})
	s.dispatcher.Register(mtypes.QueryType_TestTxByPosition, &queryplan.TestTxByPositionQueryPlan{})
	s.dispatcher.Register(mtypes.QueryType_TestHashesByHeight, &queryplan.TestHashesByHeightQueryPlan{})
	s.dispatcher.Register(mtypes.QueryType_TestPositionByHash, &queryplan.TestPositionByHashQueryPlan{})
	s.dispatcher.Register(mtypes.QueryType_TestHeightByHash, &queryplan.TestHeightByHashQueryPlan{})
	s.dispatcher.Register(mtypes.QueryType_TestReceiptsByHeight, &queryplan.TestReceiptsByHeightQueryPlan{})
	s.dispatcher.Register(mtypes.QueryType_TestReceiptByPosition, &queryplan.TestReceiptByPositionQueryPlan{})
	s.dispatcher.Register(mtypes.QueryType_TestHeightByHashOrNumber, &queryplan.TestGetHeightByHashOrNumberQueryPlan{
		GetLastHeight: func() uint64 {
			return s.lastHeight
		},
	})
	s.dispatcher.Register(mtypes.QueryType_TestGetTransactionByPosition, &queryplan.TestGetTransactionByPositionQueryPlan{})
	s.dispatcher.Register(mtypes.QueryType_TestRpcBlockSubPlan, &queryplan.TestRpcBlockQueryPlan{})

	//-------------------------
	s.dispatcher.Register(mtypes.QueryType_Receipt_Eth, &queryplan.ReceiptQueryPlan{})

	s.dispatcher.Register(mtypes.QueryType_Block_Receipts,
		&queryplan.BlockReceiptsQueryPlan{
			GetLastHeight: func() uint64 {
				return s.lastHeight
			},
		},
	)

	s.dispatcher.Register(mtypes.QueryType_Transaction, &queryplan.TransactionQueryPlan{
		GetChainId: func() *big.Int {
			return s.chainID
		},
	})

	s.dispatcher.Register(mtypes.QueryType_Block_Eth,
		&queryplan.BlockEthQueryPlan{
			GetQueryHeight: s.getQueryHeight,
			GetChainId: func() *big.Int {
				return s.chainID
			},
		},
	)

	s.dispatcher.Register(mtypes.QueryType_HeaderByNumber,
		&queryplan.HeaderByNumberQueryPlan{
			GetQueryHeight: s.getQueryHeight,
			GetChainId: func() *big.Int {
				return s.chainID
			},
		},
	)

	s.dispatcher.Register(mtypes.QueryType_HeaderByHash, &queryplan.HeaderByHashQueryPlan{
		GetChainId: func() *big.Int {
			return s.chainID
		},
	})
	s.dispatcher.Register(mtypes.QueryType_BlocByHash, &queryplan.BlockByHashQueryPlan{
		GetChainId: func() *big.Int {
			return s.chainID
		},
	})

	s.dispatcher.Register(mtypes.QueryType_Logs,
		&queryplan.InlineQueryPlan{
			Fn: func(ctx *query.QueryContext) (interface{}, error) {
				request := ctx.Req.(*evm.FilterQuery)
				return s.caches.Query(*request), nil
			},
		},
	)

	s.dispatcher.Register(mtypes.QueryType_TxNumsByHash, &queryplan.TxNumsByHashQueryPlan{})

	s.dispatcher.Register(mtypes.QueryType_TxNumsByNumber,
		&queryplan.TxNumsByNumberQueryPlan{
			GetQueryHeight: s.getQueryHeight,
		},
	)

	s.dispatcher.Register(mtypes.QueryType_TxByHashAndIdx,
		&queryplan.TxByHashAndIdxQueryPlan{
			GetChainId: func() *big.Int {
				return s.chainID
			},
		},
	)
	s.dispatcher.Register(mtypes.QueryType_TxByNumberAndIdx,
		&queryplan.TxByNumberAndIdxQueryPlan{
			GetChainId: func() *big.Int {
				return s.chainID
			},
			GetQueryHeight: s.getQueryHeight,
		},
	)

	s.dispatcher.Register(mtypes.QueryType_TxMessage,
		&queryplan.TxMessageQueryPlan{
			GetChainId: func() *big.Int {
				return s.chainID
			},
		},
	)
}
func (rs *Storage) getQueryHeight(number int64) int64 {
	queryHeight := int64(0)
	if number < 0 {
		queryHeight = int64(rs.lastHeight)
	} else {
		queryHeight = number
	}
	return queryHeight
}
