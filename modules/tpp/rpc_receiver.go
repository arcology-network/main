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

package tpp

import (
	"context"
	"sync"

	tppTypes "github.com/arcology-network/main/modules/tpp/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
)

var (
	rpcInstance actor.IWorkerEx
	initRpcOnce sync.Once
)

type RpcReceiver struct {
	actor.WorkerThread
}

// return a Subscriber struct
func NewRpcReceiver(concurrency int, groupid string) actor.IWorkerEx {
	initRpcOnce.Do(func() {

		in := RpcReceiver{}
		in.Set(concurrency, groupid)

		rpcInstance = &in
	})
	return rpcInstance
}

func (rr *RpcReceiver) Inputs() ([]string, bool) {
	return []string{}, false
}

func (rr *RpcReceiver) Outputs() map[string]int {
	return map[string]int{
		actor.MsgCheckingTxs: 100,
	}
}

func (rr *RpcReceiver) OnStart() {
}

func (rr *RpcReceiver) OnMessageArrived(msgs []*actor.Message) error {
	return nil
}

func (rr *RpcReceiver) ReceivedTransactionFromRpc(ctx context.Context, args *mtypes.RawTransactionArgs, reply *mtypes.RawTransactionReply) error {
	if rr.LatestMessage == nil {
		rr.LatestMessage = actor.NewMessage()
	}
	logid := rr.AddLog(log.LogLevel_Debug, "to StandardTransactions")
	interLog := rr.GetLogger(logid)

	pack := tppTypes.NewPack([][]byte{args.Tx}, args.Src, true, rr.Concurrency, interLog)

	rr.MsgBroker.Send(actor.MsgCheckingTxs, pack)

	reply.TxHash = <-pack.TxHashChan

	return nil
}
