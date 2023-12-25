package receipthashing

import (
	"time"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/evm/crypto"
	"go.uber.org/zap"
)

const (
	actCostgas uint64 = 21000
)

type CalculateRoothash struct {
	actor.WorkerThread
}

// return a Subscriber struct
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
	var selectedReceipts *map[evmCommon.Hash]*types.ReceiptHash
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgInclusive:
			inclusiveList = v.Data.(*types.InclusiveList)
			isnil, err := cr.IsNil(inclusiveList, "inclusiveList")
			if isnil {
				return err
			}
		case actor.MsgSelectedReceiptsHash:
			selectedReceipts = v.Data.(*map[evmCommon.Hash]*types.ReceiptHash)
			isnil, err := cr.IsNil(selectedReceipts, "selectedReceipts")
			if isnil {
				return err
			}
		}
	}
	cr.CheckPoint("start calculate rcpthash")
	hash, gas := cr.gatherReceipts(inclusiveList, selectedReceipts)
	cr.MsgBroker.Send(actor.MsgRcptHash, &hash)
	cr.MsgBroker.Send(actor.MsgGasUsed, gas)
	cr.CheckPoint("rcpthash calculate completed")
	return nil
}

func (cr *CalculateRoothash) gatherReceipts(inclusiveList *types.InclusiveList, receipts *map[evmCommon.Hash]*types.ReceiptHash) (evmCommon.Hash, uint64) {
	begintime := time.Now()
	datas := make([][]byte, 0, len(inclusiveList.HashList))
	var gasused uint64 = 0

	nilroot := evmCommon.Hash{}
	if inclusiveList == nil || receipts == nil {
		return nilroot, 0
	}

	for i, hash := range inclusiveList.HashList {
		if inclusiveList.Successful[i] {

			if rcpt, ok := (*receipts)[*hash]; ok {
				if rcpt != nil {
					datas = append(datas, rcpt.Receipthash.Bytes())
					gasused += rcpt.GasUsed
				}
			}
		}
	}

	roothash := evmCommon.Hash{}
	if len(datas) > 0 {
		// src := bytes.Join(datas, []byte(""))
		// totallen := len(src)
		// roothashbytes, err := mhasher.Roothash(datas, mhasher.HashType_256)

		// roothashbytes := crypto.Keccak256(common.Flatten(datas))

		// if err != nil {
		// 	cr.AddLog(log.LogLevel_Error, "make roothash err ", zap.String("err", err.Error()))
		// 	return nilroot, 0
		// }
		roothash = evmCommon.BytesToHash(crypto.Keccak256(common.Flatten(datas)))
		cr.AddLog(log.LogLevel_Info, "calculate recept roothash ", zap.Duration("times", time.Now().Sub(begintime)))
	}
	return roothash, gasused
}
