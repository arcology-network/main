package receipthashing

import (
	"time"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/mhasher"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/component-lib/log"
	"go.uber.org/zap"
)

const (
	actCostgas uint64 = 21000
)

type CalculateRoothash struct {
	actor.WorkerThread
}

//return a Subscriber struct
func NewCalculateRoothash(concurrency int, groupid string) actor.IWorkerEx {
	cr := CalculateRoothash{}
	cr.Set(concurrency, groupid)
	return &cr
}

func (cr *CalculateRoothash) Inputs() ([]string, bool) {
	return []string{actor.MsgSelectedReceiptsHash, actor.MsgInclusive}, true
}

func (cr *CalculateRoothash) Outputs() map[string]int {
	return map[string]int{
		actor.MsgRcptHash: 1,
		actor.MsgGasUsed:  1,
	}
}

func (cr *CalculateRoothash) OnStart() {
}

func (cr *CalculateRoothash) Stop() {

}

func (cr *CalculateRoothash) OnMessageArrived(msgs []*actor.Message) error {
	var inclusiveList *types.InclusiveList
	var selectedReceipts *map[ethCommon.Hash]*types.ReceiptHash
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgInclusive:
			inclusiveList = v.Data.(*types.InclusiveList)
			isnil, err := cr.IsNil(inclusiveList, "inclusiveList")
			if isnil {
				return err
			}
		case actor.MsgSelectedReceiptsHash:
			selectedReceipts = v.Data.(*map[ethCommon.Hash]*types.ReceiptHash)
			isnil, err := cr.IsNil(selectedReceipts, "selectedReceipts")
			if isnil {
				return err
			}
		}
	}
	cr.AddLog(log.LogLevel_Info, "CalculateRoothash start calculate")
	hash, gas := cr.gatherReceipts(inclusiveList, selectedReceipts)
	cr.MsgBroker.Send(actor.MsgRcptHash, &hash)
	cr.MsgBroker.Send(actor.MsgGasUsed, gas)
	return nil
}

func (cr *CalculateRoothash) gatherReceipts(inclusiveList *types.InclusiveList, receipts *map[ethCommon.Hash]*types.ReceiptHash) (ethCommon.Hash, uint64) {
	begintime := time.Now()
	datas := make([][]byte, len(inclusiveList.HashList))
	var gasused uint64 = 0
	nilroot := ethCommon.Hash{}
	if inclusiveList == nil || receipts == nil {
		return nilroot, 0
	}
	for i, hash := range inclusiveList.HashList {
		if inclusiveList.Successful[i] {

			if rcpt, ok := (*receipts)[*hash]; ok {
				if rcpt != nil {
					datas[i] = rcpt.Receipthash.Bytes()
					gasused += rcpt.GasUsed
				}
			}
		}
	}
	roothash := ethCommon.Hash{}
	if len(datas) > 0 {
		// src := bytes.Join(datas, []byte(""))
		// totallen := len(src)
		roothashbytes, err := mhasher.Roothash(datas, mhasher.HashType_256)
		if err != nil {
			cr.AddLog(log.LogLevel_Error, "make roothash err ", zap.String("err", err.Error()))
			return nilroot, 0
		}
		roothash = ethCommon.BytesToHash(roothashbytes)
		cr.AddLog(log.LogLevel_Info, "calculate recept roothash ", zap.Duration("times", time.Now().Sub(begintime)))
	}
	return roothash, gasused
}
