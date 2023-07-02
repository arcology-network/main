package gateway

import (
	"context"
	"sync"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	evmCommon "github.com/arcology-network/evm/common"
	gatewayTypes "github.com/arcology-network/main/modules/gateway/types"
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

func (lr *LocalReceiver) ReceivedTransactions(ctx context.Context, args *types.SendTransactionArgs, reply *types.SendTransactionReply) error {
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

func (lr *LocalReceiver) SendRawTransaction(ctx context.Context, args *types.RawTransactionArgs, reply *types.RawTransactionReply) error {
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
