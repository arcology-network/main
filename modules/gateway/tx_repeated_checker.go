package gateway

import (
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/main/components/storage"
	gatewayTypes "github.com/arcology-network/main/modules/gateway/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type TxRepeatedChecker struct {
	actor.WorkerThread
	checklist *storage.CheckedList
	waits     int64
	maxSize   int
}

// return a Subscriber struct
func NewTxRepeatedChecker(concurrency int, groupid string) actor.IWorkerEx {
	receiver := TxRepeatedChecker{}
	receiver.Set(concurrency, groupid)
	return &receiver
}

func (r *TxRepeatedChecker) Inputs() ([]string, bool) {
	return []string{
		actor.MsgTxLocalsUnChecked,
		actor.MsgTxBlocks,
	}, false
}

func (r *TxRepeatedChecker) Outputs() map[string]int {
	return map[string]int{
		actor.MsgCheckedTxs: 100,
		actor.MsgTxLocals:   100,
	}
}

func (r *TxRepeatedChecker) Config(params map[string]interface{}) {
	r.waits = int64(params["wait_seconds"].(float64))
	r.maxSize = int(params["max_txs_num"].(float64))
}

func (r *TxRepeatedChecker) OnStart() {
	r.checklist = storage.NewCheckList(r.waits)
}

func (r *TxRepeatedChecker) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgTxLocalsUnChecked:
			data := v.Data.(*gatewayTypes.TxsPack)
			r.checkRepeated(data, types.TxFrom_Local)
		case actor.MsgTxBlocks:
			pack := &gatewayTypes.TxsPack{
				Txs: v.Data.(*types.IncomingTxs),
			}
			r.checkRepeated(pack, types.TxFrom_Block)
		}
	}
	return nil
}

func (r *TxRepeatedChecker) checkRepeated(txspack *gatewayTypes.TxsPack, from byte) {
	txs := txspack.Txs.Txs
	txLen := len(txs)
	checkedTxs := make([][]byte, 0, txLen)
	logid := r.AddLog(log.LogLevel_Debug, "checkRepeated")
	interLog := r.GetLogger(logid)
	bypassRepeatCheck := txspack.Txs.Src.BypassRepeatCheck()
	for i := range txs {
		isExist := r.checklist.ExistTx(txs[i], from, interLog)
		if !isExist || bypassRepeatCheck {
			tx := txs[i]
			sendingTx := make([]byte, len(tx)+1)
			bz := 0
			bz += copy(sendingTx[bz:], []byte{from})
			bz += copy(sendingTx[bz:], tx)

			checkedTxs = append(checkedTxs, sendingTx)
		}
	}

	//to other node with consensus
	if from == types.TxFrom_Local {
		r.MsgBroker.Send(actor.MsgTxLocals, txs)
	}

	//to tpp with rpc
	if txspack.TxHashChan != nil {
		go func() {
			if len(checkedTxs) > 0 {
				response := mtypes.RawTransactionReply{}
				intf.Router.Call("tpp", "ReceivedTransactionFromRpc", &mtypes.RawTransactionArgs{
					Tx:  checkedTxs[0],
					Src: txspack.Txs.Src,
				}, &response)
				txspack.TxHashChan <- response.TxHash.(evmCommon.Hash)
			} else {
				txspack.TxHashChan <- evmCommon.Hash{}
			}
		}()
		return
	}
	//to tpp with kafka
	sendingTxs := make([][]byte, 0, r.maxSize)
	for i := range checkedTxs {
		if len(sendingTxs) >= r.maxSize {
			r.MsgBroker.Send(actor.MsgCheckedTxs, &types.IncomingTxs{
				Txs: sendingTxs,
				Src: txspack.Txs.Src,
			})
			sendingTxs = make([][]byte, 0, r.maxSize)
		} else {
			sendingTxs = append(sendingTxs, checkedTxs[i])
		}
	}
	if len(sendingTxs) > 0 {
		r.MsgBroker.Send(actor.MsgCheckedTxs, &types.IncomingTxs{
			Txs: sendingTxs,
			Src: txspack.Txs.Src,
		})
	}

}
