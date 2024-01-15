package tpp

import (
	"context"
	"sync"

	"github.com/arcology-network/common-lib/types"
	tppTypes "github.com/arcology-network/main/modules/tpp/types"
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

func (rr *RpcReceiver) ReceivedTransactionFromRpc(ctx context.Context, args *types.RawTransactionArgs, reply *types.RawTransactionReply) error {
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
