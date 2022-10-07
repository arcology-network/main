package tpp

import (
	"github.com/HPISTechnologies/common-lib/common"
	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/component-lib/log"
	tppTypes "github.com/HPISTechnologies/main/modules/tpp/types"
	"go.uber.org/zap"
)

type TxReceiver struct {
	actor.WorkerThread
}

//return a Subscriber struct
func NewTxReceiver(concurrency int, groupid string) actor.IWorkerEx {
	receiver := TxReceiver{}
	receiver.Set(concurrency, groupid)

	return &receiver
}

func (r *TxReceiver) Inputs() ([]string, bool) {
	return []string{actor.MsgCheckedTxs}, false
}

func (r *TxReceiver) Outputs() map[string]int {
	return map[string]int{
		actor.MsgCheckingTxs: 10,
	}
}

func (r *TxReceiver) OnStart() {
}

func (r *TxReceiver) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgCheckedTxs:
			data := v.Data.([][]byte)
			r.parallelSendTxs(data)
		}
	}

	return nil
}

func (r *TxReceiver) parallelSendTxs(rawtxs [][]byte) {
	txLen := len(rawtxs)
	checks := make([]*tppTypes.CheckingTx, txLen)

	logid := r.AddLog(log.LogLevel_Debug, "start parallelSend Txs")
	interLog := r.GetLogger(logid)
	common.ParallelWorker(txLen, r.Concurrency, tppTypes.CheckingTxHashWorker, rawtxs, &checks, interLog)

	r.AddLog(log.LogLevel_Debug, "parallelSendTxs completed <<<<<<<<<<", zap.Int("txLen", len(checks)))

	pack := tppTypes.CheckingTxsPack{
		Txs: checks,
	}

	r.MsgBroker.Send(actor.MsgCheckingTxs, &pack)
}
