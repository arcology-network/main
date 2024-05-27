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
	"context"
	"sync"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	gatewayTypes "github.com/arcology-network/main/modules/gateway/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type LocalReceiver struct {
	actor.WorkerThread
}

var (
	lrSingleton actor.IWorkerEx
	initOnce    sync.Once
)

// return a Subscriber struct
func NewLocalReceiver(concurrency int, groupid string) actor.IWorkerEx {
	initOnce.Do(func() {
		in := LocalReceiver{}
		in.Set(concurrency, groupid)
		lrSingleton = &in
	})
	return lrSingleton
}

func (lr *LocalReceiver) Inputs() ([]string, bool) {
	return []string{}, false
}

func (lr *LocalReceiver) Outputs() map[string]int {
	return map[string]int{
		actor.MsgTxLocalsUnChecked: 100,
	}
}

func (lr *LocalReceiver) OnStart() {
}

func (lr *LocalReceiver) OnMessageArrived(msgs []*actor.Message) error {
	return nil
}

func (lr *LocalReceiver) ReceivedTransactions(ctx context.Context, args *mtypes.SendTransactionArgs, reply *mtypes.SendTransactionReply) error {
	txLen := len(args.Txs)
	checkingtxs := make([][]byte, txLen)
	common.ParallelWorker(txLen, lr.Concurrency, lr.txWorker, args.Txs, &checkingtxs)
	txsPack := gatewayTypes.TxsPack{
		Txs: &types.IncomingTxs{
			Txs: checkingtxs,
			Src: types.NewTxSource(types.TxSourceLocal, "frontend"),
		},
	}
	lr.MsgBroker.Send(actor.MsgTxLocalsUnChecked, &txsPack)

	reply.Status = 0
	return nil
}

func (lr *LocalReceiver) SendRawTransaction(ctx context.Context, args *mtypes.RawTransactionArgs, reply *mtypes.RawTransactionReply) error {
	txLen := 1
	checkingtxs := make([][]byte, txLen)
	common.ParallelWorker(txLen, lr.Concurrency, lr.txWorker, [][]byte{args.Tx}, &checkingtxs)
	txsPack := gatewayTypes.TxsPack{
		Txs: &types.IncomingTxs{
			Txs: checkingtxs,
			Src: types.NewTxSource(types.TxSourceLocal, "ethapi"),
		},
		TxHashChan: make(chan evmCommon.Hash, 1),
	}
	lr.MsgBroker.Send(actor.MsgTxLocalsUnChecked, &txsPack)

	hash := <-txsPack.TxHashChan

	reply.TxHash = evmCommon.BytesToHash(hash.Bytes())
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
