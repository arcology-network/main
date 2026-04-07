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

package pool

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/arcology-network/common-lib/types"
	mtypes "github.com/arcology-network/main/types"
	statestore "github.com/arcology-network/state-engine"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

type AggrSelector struct {
	maxReap      int
	obsoleteTime uint64
	closeCheck   bool
	pool         *Pool
	state        int
	height       uint64

	opAdaptor *OpAdaptor
	chainID   *big.Int

	ctx *actor.ExecutionContext
}

const (
	poolStateClean = iota
	poolStateReap
	poolStateCherryPick
	resultCollect
)

// return a Subscriber struct
func NewAggrSelector() actor.Business {
	rpcInstance := &AggrSelector{
		state: poolStateClean,
	}

	return rpcInstance
}

func (a *AggrSelector) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgNonceReady,
		scommon.MsgMessager,
		scommon.MsgReapCommand,
		scommon.MsgReapinglist,
		scommon.MsgSelectedReceipts,
		scommon.MsgPendingBlock,
		scommon.MsgInitialization,
		scommon.MsgOpCommand,
	}, false
}

func (a *AggrSelector) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgMessagersReaped: 1,
		scommon.MsgMetaBlock:       1,
		scommon.MsgSelectedTxInfo:  1,
		scommon.MsgBlockParams:     1,
		scommon.MsgWithDrawHash:    1,
		scommon.MsgSignerType:      1,
		scommon.MsgOpCommand:       1,
	}
}

func (a *AggrSelector) Config(params map[string]interface{}) {
	a.maxReap = params["max_reap_size"].(int)
	a.obsoleteTime = uint64(params["obsolete_time"].(int))
	if _, ok := params["close_check"]; ok {
		a.closeCheck = params["close_check"].(bool)
	}
	a.chainID = params["chain_id"].(*big.Int)
	a.opAdaptor = NewOpAdaptor(a.maxReap, a.chainID)
}

func (a *AggrSelector) reap(ctx *actor.ActionContext, height uint64) {
	reaped := a.pool.Reap(a.opAdaptor.ReapSize)
	a.send(ctx, reaped, true, height)
	a.ChangeState(ctx, poolStateCherryPick, "poolStateCherryPick")
}

func (a *AggrSelector) returnResult(ctx *actor.ActionContext, result *mtypes.BlockResult) {
	if !mtypes.RunAsL1 {
		a.ctx.SendRpcResponse("", result)
	}

	a.ChangeState(ctx, poolStateClean, "poolStateClean")
}

func (a *AggrSelector) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgNonceReady, a.ReceivedNonceReady)
	reg.Register(scommon.MsgInitialization, a.ReceivedInitialization)
	reg.Register(scommon.MsgMessager, a.ReceivedMessage)
	reg.Register(scommon.MsgReapCommand, a.ReceivedReapCommand)
	reg.Register(scommon.MsgOpCommand, a.ReceivedOpCommand)
	reg.Register("ReceivedMessages", a.ReceivedMessages)
	reg.Register(scommon.MsgReapinglist, a.ReceivedReapinglist)
	reg.Register(scommon.MsgSelectedReceipts, a.ReceivedSelectedReceipts)
	reg.Register(scommon.MsgPendingBlock, a.ReceivedPendingBlock)

	reg.Register("Query", a.Query)
}
func (a *AggrSelector) ReceivedOpCommand(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	oprequest := msg.Data.(*mtypes.OpRequest)
	ctx.ExecCtx.LogDebug("oprequest received", logger.F("oprequest.Transactions", len(oprequest.Transactions)), logger.F("oprequest.Withdrawals", len(oprequest.Withdrawals)))
	if a.opAdaptor.AddOpCommand(oprequest.Transactions, oprequest.Withdrawals) {
		a.reap(ctx, a.height)
	}
	return nil
}
func (a *AggrSelector) ReceivedNonceReady(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	a.nonceReady(ctx, msg.Data.(*statestore.StateStore), msg.Height)
	return nil
}
func (a *AggrSelector) ReceivedInitialization(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	initialization := msg.Data.(*mtypes.Initialization)
	a.opAdaptor.SetConfig(initialization.ChainConfig)
	a.opAdaptor.ChangeSigner(msg.Height)

	a.nonceReady(ctx, initialization.Store, msg.Height)
	ctx.ExecCtx.LogDebug("change into poolStateReap,ready")
	return nil
}
func (a *AggrSelector) nonceReady(ctx *actor.ActionContext, store *statestore.StateStore, heght uint64) {
	if a.pool == nil {
		a.pool = NewPool(store, a.obsoleteTime, a.closeCheck)
	} else {
		a.pool.Clean(heght)
		ctx.ExecCtx.LogDebug("Clear pool", logger.F("height", heght))
	}
	a.opAdaptor.Reset()

	a.ChangeState(ctx, poolStateReap, "poolStateReap")
	a.height = heght + 1
}
func (a *AggrSelector) ChangeState(ctx *actor.ActionContext, state int, stateName string) error {
	a.state = state
	ctx.ExecCtx.LogDebug("****** " + ctx.ExecCtx.WorkCtx.BusinassName + " state change into " + stateName)
	return nil
}
func (a *AggrSelector) ReceivedMessage(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]

	switch a.state {
	case poolStateReap:
		msgs := msg.Data.(*types.StdTransactionPack)
		a.pool.Add(msgs.Txs, msgs.Src, a.height)
		ctx.ExecCtx.LogDebug("ReceivedMessage", logger.F("msgs.Txs", len(msgs.Txs)))
	case poolStateCherryPick:
		msgs := msg.Data.(*types.StdTransactionPack)
		reaped := a.pool.Add(msgs.Txs, msgs.Src, a.height)
		ctx.ExecCtx.LogDebug("ReceivedMessage", logger.F("msgs.Txs", len(msgs.Txs)))
		if reaped != nil {
			a.send(ctx, reaped, false, a.height)

			a.ChangeState(ctx, resultCollect, "resultCollect")
		}
	}
	return nil
}

func (a *AggrSelector) ReceivedReapCommand(ctx *actor.ActionContext) error {
	// msg := ctx.Messages[0]
	if mtypes.RunAsL1 {
		ctx.ExecCtx.Send(scommon.MsgOpCommand, &mtypes.OpRequest{
			Withdrawals:  evmTypes.Withdrawals{},
			Transactions: []*types.StandardTransaction{},
		}, a.height)
		ctx.ExecCtx.Send(scommon.MsgBlockParams, &mtypes.BlockParams{
			Random:     evmCommon.Hash{},
			BeaconRoot: &evmCommon.Hash{},
			Times:      0,
		}, a.height)
		ctx.ExecCtx.Send(scommon.MsgWithDrawHash, &evmTypes.EmptyWithdrawalsHash, a.height)
	}
	if a.opAdaptor.AddReapCommand() {
		a.reap(ctx, a.height)
	}
	return nil
}

func (a *AggrSelector) ReceivedReapinglist(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	list := msg.Data.(*types.ReapingList).List
	ctx.ExecCtx.LogDebug("ReceivedReapinglist", logger.F("reapeds", len(list)))

	reaped := a.pool.CherryPick(a.opAdaptor.ClipReapList(list))
	if reaped != nil {
		a.send(ctx, reaped, false, a.height)
		a.ChangeState(ctx, resultCollect, "resultCollect")
	}
	return nil
}

func (a *AggrSelector) ReceivedSelectedReceipts(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	receipts := msg.Data.([]*evmTypes.Receipt)
	if ok, result := a.opAdaptor.AddReceipts(receipts); ok {
		a.returnResult(ctx, result)
	}
	return nil
}

func (a *AggrSelector) ReceivedPendingBlock(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	block := msg.Data.(*mtypes.MonacoBlock)
	if ok, result := a.opAdaptor.AddBlock(block); ok {
		a.returnResult(ctx, result)
	}
	return nil
}

func (a *AggrSelector) send(ctx *actor.ActionContext, reaped []*types.StandardTransaction, isProposer bool, height uint64) {
	ctx.ExecCtx.LogDebug("reap end", logger.F("reapeds", len(reaped)), logger.F("isProposer", isProposer))
	if isProposer {
		hashes := make([]evmCommon.Hash, len(reaped))
		for i := range hashes {
			hashes[i] = reaped[i].TxHash
		}

		ctx.ExecCtx.Send(scommon.MsgMetaBlock, &mtypes.MetaBlock{
			Txs:      [][]byte{},
			Hashlist: a.opAdaptor.AppendList(hashes),
		}, height)
	} else {
		msgs, transactions, txs := a.opAdaptor.ReapEnd(reaped)
		sendMsgs := make([]*types.StandardMessage, len(msgs))
		hashList := make([]evmCommon.Hash, len(msgs))
		for i := range msgs {
			sendMsgs[i] = &types.StandardMessage{
				ID:     uint64(i + 1),
				TxHash: msgs[i].TxHash,
				Native: msgs[i].NativeMessage,
				Source: msgs[i].Source,
			}
			sendMsgs[i].Native.SkipAccountChecks = true
			hashList[i] = msgs[i].TxHash
		}
		ctx.ExecCtx.Send(scommon.MsgMessagersReaped, sendMsgs, height)
		ctx.ExecCtx.LogInfo("send messagersReaped", logger.F("msgs", len(msgs)))

		txhash := evmTypes.EmptyTxsHash
		if len(transactions) > 0 {
			txhash = evmTypes.DeriveSha(evmTypes.Transactions(transactions), trie.NewStackTrie(nil))
		}
		ctx.ExecCtx.Send(scommon.MsgSelectedTxInfo, &mtypes.SelectedTxsInfo{
			Txhash:   txhash,
			Txs:      txs,
			HashList: hashList,
		}, height)
		ctx.ExecCtx.Send(scommon.MsgSignerType, a.opAdaptor.SignerType, height)
		ctx.ExecCtx.LogInfo("send selectedtx", logger.F("txhash", fmt.Sprintf("%x", txhash)))
	}
}

func (a *AggrSelector) GetFSMRules() map[int]actor.FSMRule {
	return map[int]actor.FSMRule{
		poolStateClean: {Accept: []string{
			scommon.MsgNonceReady,     // <- nonce-url
			scommon.MsgInitialization, // <- storage.initilizator
		}},
		poolStateReap: {Accept: []string{ // <- proposer
			scommon.MsgMessager,    // <- tpp
			scommon.MsgReapCommand, // <- consensus
			scommon.MsgOpCommand,   // <- op  from rpc or self
		}},
		poolStateCherryPick: {Accept: []string{ // <- non proposer
			scommon.MsgMessager,    // <- tpp
			scommon.MsgReapinglist, // <- consensus
		}},
		resultCollect: {Accept: []string{
			scommon.MsgSelectedReceipts, // <- receipt aggregator
			scommon.MsgPendingBlock,     // <- core
		}},
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

func (a *AggrSelector) RpcConfig() (string, int) {
	return "pool", 20
}
func (a *AggrSelector) ReceivedMessages(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*mtypes.OpRequest)
	ctx.ExecCtx.Send(scommon.MsgBlockParams, request.BlockParam, a.height)
	ctx.ExecCtx.Send(scommon.MsgOpCommand, request, a.height)

	var withdrawalsHash *evmCommon.Hash
	if request.Withdrawals == nil {
		withdrawalsHash = nil
	} else if len(request.Withdrawals) == 0 {
		withdrawalsHash = &evmTypes.EmptyWithdrawalsHash
	} else {
		h := evmTypes.DeriveSha(evmTypes.Withdrawals(request.Withdrawals), trie.NewStackTrie(nil))
		withdrawalsHash = &h
	}
	ctx.ExecCtx.Send(scommon.MsgWithDrawHash, withdrawalsHash, a.height)

	a.ctx = ctx.ExecCtx.Fork()

	return nil
}
func (a *AggrSelector) Query(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*mtypes.QueryRequest)
	switch request.QueryType {
	case mtypes.QueryType_Transaction:
		hash := request.Data.(evmCommon.Hash)
		st := a.pool.QueryByHash(evmCommon.BytesToHash(hash.Bytes()))
		if st == nil {
			ctx.ExecCtx.SendRpcResponse("hash not found", &mtypes.QueryResult{
				Data: nil,
			})
			return errors.New("hash not found")
		}
		txReal := st.TxRawData[1:]
		otx := new(evmTypes.Transaction)
		if err := otx.UnmarshalBinary(txReal); err != nil {
			ctx.ExecCtx.SendRpcResponse("tx decode err", &mtypes.QueryResult{
				Data: nil,
			})
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
		ctx.ExecCtx.SendRpcResponse("", &mtypes.QueryResult{
			Data: &transaction,
		})
	default:
		ctx.ExecCtx.SendRpcResponse("query type not found", &mtypes.QueryResult{
			Data: nil,
		})
	}
	return nil
}
