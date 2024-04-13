package pool

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"

	"github.com/arcology-network/common-lib/types"
	eucommon "github.com/arcology-network/eu/common"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/storage-committer/storage/statestore"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"go.uber.org/zap"
)

type AggrSelector struct {
	actor.WorkerThread

	maxReap      int
	obsoleteTime uint64
	closeCheck   bool
	pool         *Pool
	state        int
	height       uint64

	opAdaptor *OpAdaptor
	chainID   *big.Int
	resultch  chan *mtypes.BlockResult

	noOp bool //no opnode
}

const (
	poolStateClean = iota
	poolStateReap
	poolStateCherryPick
	resultCollect
	poolStateInit
)

var (
	rpcInstance actor.IWorkerEx
	initRpcOnce sync.Once
)

// return a Subscriber struct
func NewAggrSelector(concurrency int, groupid string) actor.IWorkerEx {

	initRpcOnce.Do(func() {
		rpcInstance = &AggrSelector{
			state:    poolStateInit,
			resultch: make(chan *mtypes.BlockResult, 1),
		}
		rpcInstance.(*AggrSelector).Set(concurrency, groupid)
	})

	return rpcInstance
}

func (a *AggrSelector) Inputs() ([]string, bool) {
	return []string{
		actor.MsgNonceReady,
		actor.MsgMessager,
		actor.MsgReapCommand,
		actor.MsgReapinglist,
		actor.MsgOpCommand,
		actor.MsgSelectedReceipts,
		actor.MsgPendingBlock,
		actor.MsgChainConfig,
	}, false
}

func (a *AggrSelector) Outputs() map[string]int {
	return map[string]int{
		actor.MsgMessagersReaped: 1,
		actor.MsgMetaBlock:       1,
		actor.MsgSelectedTx:      1,
		actor.MsgOpCommand:       1,
		actor.MsgBlockParams:     1,
		actor.MsgTxHash:          1,
		actor.MsgWithDrawHash:    1,
		actor.MsgSignerType:      1,
	}
}

func (a *AggrSelector) Config(params map[string]interface{}) {
	a.maxReap = int(params["max_reap_size"].(float64))
	a.obsoleteTime = uint64(params["obsolete_time"].(float64))
	if _, ok := params["close_check"]; ok {
		a.closeCheck = params["close_check"].(bool)
	}
	a.chainID = params["chain_id"].(*big.Int)
	a.opAdaptor = NewOpAdaptor(a.maxReap, a.chainID)
	a.noOp = params["no_op"].(bool)
}

func (a *AggrSelector) OnStart() {
}

func (a *AggrSelector) reap(height uint64) {
	reaped := a.pool.Reap(a.opAdaptor.ReapSize)
	a.send(reaped, true, height)
	a.state = poolStateCherryPick
	a.AddLog(log.LogLevel_Info, "Reap done, switch to poolStateCherryPick")
}

func (a *AggrSelector) returnResult(result *mtypes.BlockResult) {
	if !a.noOp {
		a.resultch <- result
	}

	a.state = poolStateClean
	a.AddLog(log.LogLevel_Info, "received all results, switch to poolStateClean")
}

func (a *AggrSelector) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch a.state {
	case poolStateInit:
		switch msg.Name {
		case actor.MsgChainConfig:
			a.opAdaptor.SetConfig(msg.Data.(*params.ChainConfig))
			a.opAdaptor.ChangeSigner(msg.Height)
			a.state = poolStateClean
		}
	case poolStateClean:
		switch msg.Name {
		case actor.MsgNonceReady:
			if a.pool == nil {
				a.pool = NewPool(msg.Data.(*statestore.StateStore), a.obsoleteTime, a.closeCheck)
			} else {
				a.pool.Clean(msg.Height)
				a.AddLog(log.LogLevel_Info, fmt.Sprintf("Clear pool on height %d", msg.Height))
			}
			a.opAdaptor.Reset()
			a.state = poolStateReap
			a.height = msg.Height + 1
		}
	case poolStateReap:
		switch msg.Name {
		case actor.MsgMessager:
			msgs := msg.Data.(*types.StdTransactionPack)
			a.pool.Add(msgs.Txs, msgs.Src, msg.Height)
		case actor.MsgReapCommand:
			if a.noOp {
				a.MsgBroker.Send(actor.MsgOpCommand, &mtypes.OpRequest{
					Withdrawals:  evmTypes.Withdrawals{},
					Transactions: []*types.StandardTransaction{},
				})
				a.MsgBroker.Send(actor.MsgBlockParams, &mtypes.BlockParams{
					Random:     evmCommon.Hash{},
					BeaconRoot: &evmCommon.Hash{},
					Times:      0,
				})
				a.MsgBroker.Send(actor.MsgWithDrawHash, &evmTypes.EmptyWithdrawalsHash)
			}

			if a.opAdaptor.AddReapCommand() {
				a.reap(msg.Height)
			}
		case actor.MsgOpCommand:
			oprequest := msg.Data.(*mtypes.OpRequest)
			a.AddLog(log.LogLevel_Debug, "oprequest received", zap.Int("oprequest.Transactions", len(oprequest.Transactions)), zap.Int("oprequest.Withdrawals", len(oprequest.Withdrawals)))
			if a.opAdaptor.AddOpCommand(oprequest.Transactions, oprequest.Withdrawals) {
				a.reap(msg.Height)
			}
		}
	case poolStateCherryPick:
		switch msg.Name {
		case actor.MsgMessager:
			msgs := msg.Data.(*types.StdTransactionPack)
			reaped := a.pool.Add(msgs.Txs, msgs.Src, msg.Height)
			if reaped != nil {
				a.send(reaped, false, a.height)
				a.state = resultCollect
				a.AddLog(log.LogLevel_Info, "Data received, switch to resultCollect")
			}
		case actor.MsgReapinglist:
			a.CheckPoint("pool received reapinglist")

			reaped := a.pool.CherryPick(a.opAdaptor.ClipReapList(msg.Data.(*types.ReapingList).List))
			if reaped != nil {
				a.send(reaped, false, msg.Height)
				a.state = resultCollect
				a.AddLog(log.LogLevel_Info, "List received, switch to resultCollect")
			}
		}
	case resultCollect:
		switch msg.Name {
		case actor.MsgSelectedReceipts:
			var receipts []*evmTypes.Receipt
			for _, item := range msg.Data.([]interface{}) {
				receipts = append(receipts, item.(*evmTypes.Receipt))
			}
			if ok, result := a.opAdaptor.AddReceipts(receipts); ok {
				a.returnResult(result)
			}

		case actor.MsgPendingBlock:
			block := msg.Data.(*mtypes.MonacoBlock)
			if ok, result := a.opAdaptor.AddBlock(block); ok {
				a.returnResult(result)
			}
		}
	}
	return nil
}

func (a *AggrSelector) send(reaped []*types.StandardTransaction, isProposer bool, height uint64) {
	a.AddLog(log.LogLevel_Debug, "reap end", zap.Int("reapeds", len(reaped)))
	if isProposer {
		hashes := make([]evmCommon.Hash, len(reaped))
		for i := range hashes {
			hashes[i] = reaped[i].TxHash
		}

		a.MsgBroker.Send(actor.MsgMetaBlock, &mtypes.MetaBlock{
			Txs:      [][]byte{},
			Hashlist: a.opAdaptor.AppendList(hashes),
		}, height)
	} else {
		msgs, transactions, txs := a.opAdaptor.ReapEnd(reaped)
		sendMsgs := make([]*eucommon.StandardMessage, len(msgs))
		for i := range msgs {
			sendMsgs[i] = &eucommon.StandardMessage{
				ID:     uint64(i + 1),
				TxHash: msgs[i].TxHash,
				Native: msgs[i].NativeMessage,
				Source: msgs[i].Source,
			}
			sendMsgs[i].Native.SkipAccountChecks = true
		}
		a.MsgBroker.Send(actor.MsgMessagersReaped, sendMsgs, height)
		a.CheckPoint("send messagersReaped", zap.Int("msgs", len(msgs)))
		txhash := evmTypes.EmptyTxsHash
		if len(transactions) > 0 {
			txhash = evmTypes.DeriveSha(evmTypes.Transactions(transactions), trie.NewStackTrie(nil))
		}
		a.MsgBroker.Send(actor.MsgSelectedTx, txs, height)
		a.MsgBroker.Send(actor.MsgTxHash, &txhash, height)
		a.MsgBroker.Send(actor.MsgSignerType, a.opAdaptor.SignerType, height)
		a.CheckPoint("send selectedtx", zap.String("txhash", fmt.Sprintf("%x", txhash)))
	}
}

func (a *AggrSelector) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		poolStateInit: {
			actor.MsgChainConfig,
		},
		poolStateClean: {
			actor.MsgNonceReady,
		},
		poolStateReap: {
			actor.MsgMessager,
			actor.MsgOpCommand,
			actor.MsgReapCommand,
		},
		poolStateCherryPick: {
			actor.MsgMessager,
			actor.MsgReapinglist,
		},
		resultCollect: {
			actor.MsgSelectedReceipts,
			actor.MsgPendingBlock,
		},
	}
}

func (a *AggrSelector) GetCurrentState() int {
	return a.state
}

func (a *AggrSelector) Height() uint64 {
	if a.height == 0 {
		return math.MaxUint64
	}
	return a.height
}

func (a *AggrSelector) ReceivedMessages(ctx context.Context, request *mtypes.OpRequest, response *mtypes.QueryResult) error {
	a.MsgBroker.Send(actor.MsgOpCommand, request, a.height)
	a.MsgBroker.Send(actor.MsgBlockParams, request.BlockParam, a.height)

	var withdrawalsHash *evmCommon.Hash
	if request.Withdrawals == nil {
		withdrawalsHash = nil
	} else if len(request.Withdrawals) == 0 {
		withdrawalsHash = &evmTypes.EmptyWithdrawalsHash
	} else {
		h := evmTypes.DeriveSha(evmTypes.Withdrawals(request.Withdrawals), trie.NewStackTrie(nil))
		withdrawalsHash = &h
	}
	a.MsgBroker.Send(actor.MsgWithDrawHash, withdrawalsHash, a.height)
	response.Data = <-a.resultch
	return nil
}

func (a *AggrSelector) Query(ctx context.Context, request *mtypes.QueryRequest, response *mtypes.QueryResult) error {
	switch request.QueryType {
	case mtypes.QueryType_Transaction:
		hash := request.Data.(evmCommon.Hash)
		st := a.pool.QueryByHash(evmCommon.BytesToHash(hash.Bytes()))
		if st == nil {
			response.Data = nil
			return errors.New("hash not found")
		}
		txReal := st.TxRawData[1:]
		otx := new(evmTypes.Transaction)
		if err := otx.UnmarshalBinary(txReal); err != nil {
			return errors.New("tx decode err")
		}

		v, s, r := otx.RawSignatureValues()
		msg := st.NativeMessage
		transaction := mtypes.RPCTransaction{

			Type:     hexutil.Uint64(evmTypes.LegacyTxType),
			From:     evmCommon.Address(msg.From),
			Gas:      hexutil.Uint64(otx.Gas()),
			GasPrice: (*hexutil.Big)(otx.GasPrice()),
			Hash:     hash,
			Input:    hexutil.Bytes(otx.Data()),
			Nonce:    hexutil.Uint64(otx.Nonce()),
			To:       (*evmCommon.Address)(msg.To),
			Value:    (*hexutil.Big)(otx.Value()),
			V:        (*hexutil.Big)(v),
			R:        (*hexutil.Big)(r),
			S:        (*hexutil.Big)(s),
		}

		response.Data = &transaction
	}
	return nil
}
