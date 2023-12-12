package tpp

import (
	"context"
	"sync"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	tppTypes "github.com/arcology-network/main/modules/tpp/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
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
	txLen := 1
	checks := make([]*tppTypes.CheckingTx, txLen)

	if rr.LatestMessage == nil {
		rr.LatestMessage = actor.NewMessage()
	}
	logid := rr.AddLog(log.LogLevel_Debug, "start parallelSend Txs")
	interLog := rr.GetLogger(logid)
	common.ParallelWorker(txLen, rr.Concurrency, tppTypes.CheckingTxHashWorker, [][]byte{args.Tx}, &checks, interLog)

	pack := tppTypes.CheckingTxsPack{
		Txs:        checks,
		Src:        args.Src,
		TxHashChan: make(chan evmCommon.Hash, 1),
	}
	rr.MsgBroker.Send(actor.MsgCheckingTxs, &pack)

	reply.TxHash = <-pack.TxHashChan

	return nil
}
