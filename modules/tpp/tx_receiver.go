package tpp

import (
	cmntyp "github.com/arcology-network/common-lib/types"
	tppTypes "github.com/arcology-network/main/modules/tpp/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
)

type TxReceiver struct {
	actor.WorkerThread
}

// return a Subscriber struct
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
			data := v.Data.(*cmntyp.IncomingTxs)
			r.processTxs(data)
		}
	}

	return nil
}

func (r *TxReceiver) processTxs(txs *cmntyp.IncomingTxs) {

	logid := r.AddLog(log.LogLevel_Debug, "start processTxs Txs")
	interLog := r.GetLogger(logid)

	pack := tppTypes.NewPack(txs.Txs, txs.Src, false, r.Concurrency, interLog)

	r.MsgBroker.Send(actor.MsgCheckingTxs, pack)
}
