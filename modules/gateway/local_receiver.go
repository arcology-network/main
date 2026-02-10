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

package gateway

import (
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type LocalReceiver struct {
}

// return a Subscriber struct
func NewLocalReceiver() actor.Business {
	in := LocalReceiver{}
	return &in
}

func (lr *LocalReceiver) Inputs() ([]string, bool) {
	return []string{scommon.MsgRpcHash}, false
}

func (lr *LocalReceiver) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgTxLocalsUnChecked: 100,
	}
}

func (lr *LocalReceiver) RpcConfig() (string, int) {
	return "gateway", 20
}

func (lr *LocalReceiver) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("ReceivedTransactions", lr.ReceivedTransactions)
	reg.Register("SendRawTransaction", lr.SendRawTransaction)
	reg.Register(scommon.MsgRpcHash, lr.ReturnRpcHash)
}

func (lr *LocalReceiver) ReceivedTransactions(ctx *actor.ActionContext) error {
	args := ctx.RPC.Request.(*mtypes.SendTransactionArgs)
	txLen := len(args.Txs)
	checkingtxs := make([][]byte, txLen)
	common.ParallelWorker(txLen, ctx.ExecCtx.Concurrency(), lr.txWorker, args.Txs, &checkingtxs)
	txsPack := types.IncomingTxs{
		Txs:       checkingtxs,
		Src:       types.NewTxSource(types.TxSourceLocal, "ethapibatch"),
		RequestID: "",
	}

	ctx.ExecCtx.Send(scommon.MsgTxLocalsUnChecked, &txsPack)
	ctx.ExecCtx.SendRpcResponse("", &mtypes.SendTransactionReply{
		Status: 0,
	})

	return nil
}

func (lr *LocalReceiver) SendRawTransaction(ctx *actor.ActionContext) error {
	args := ctx.RPC.Request.(*mtypes.RawTransactionArgs)
	txLen := 1
	ctx.ExecCtx.LogDebug("LocalReceiver.SendRawTransaction")
	checkingtxs := make([][]byte, txLen)
	common.ParallelWorker(txLen, ctx.ExecCtx.Concurrency(), lr.txWorker, [][]byte{args.Tx}, &checkingtxs)
	txsPack := types.IncomingTxs{
		Txs:       checkingtxs,
		Src:       types.NewTxSource(types.TxSourceLocal, "ethapi"),
		RequestID: ctx.ExecCtx.Current.ReqID,
	}

	ctx.ExecCtx.Send(scommon.MsgTxLocalsUnChecked, &txsPack)

	return nil
}
func (lr *LocalReceiver) ReturnRpcHash(ctx *actor.ActionContext) error {
	hash := ctx.Messages[0].Data.(evmCommon.Hash)

	ctx.ExecCtx.SendRpcResponse("", &mtypes.RawTransactionReply{
		TxHash: hash,
	})
	return nil
}

func (lr *LocalReceiver) txWorker(start, end, idx int, args ...interface{}) {
	txs := args[0].([]interface{})[0].([][]byte)
	streamerTxs := args[0].([]interface{})[1].(*[][]byte)

	for i := start; i < end; i++ {
		tx := txs[i]
		sendingTx := make([]byte, len(tx)+1)
		bz := 0
		bz += copy(sendingTx[bz:], []byte{types.TxType_Eth})
		bz += copy(sendingTx[bz:], tx)
		(*streamerTxs)[i] = sendingTx
	}
}
