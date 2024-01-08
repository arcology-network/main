package receipthashing

import (
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
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
	return []string{actor.MsgSelectedReceipts, actor.MsgInclusive}, true
}

func (cr *CalculateRoothash) Outputs() map[string]int {
	return map[string]int{
		actor.MsgRcptHash: 1,
		actor.MsgGasUsed:  1,
		actor.MsgBloom:    1,
	}
}

func (cr *CalculateRoothash) OnStart() {
}

func (cr *CalculateRoothash) Stop() {

}

func (cr *CalculateRoothash) OnMessageArrived(msgs []*actor.Message) error {
	var inclusiveList *types.InclusiveList
	var selectedReceipts []*evmTypes.Receipt
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgInclusive:
			inclusiveList = v.Data.(*types.InclusiveList)
			isnil, err := cr.IsNil(inclusiveList, "inclusiveList")
			if isnil {
				return err
			}
		case actor.MsgSelectedReceipts:
			for _, item := range v.Data.([]interface{}) {
				selectedReceipts = append(selectedReceipts, item.(*evmTypes.Receipt))
			}

		}
	}
	cr.CheckPoint("start calculate rcpthash")
	hash, bloom, gas := cr.gatherReceipts(inclusiveList, selectedReceipts)
	cr.MsgBroker.Send(actor.MsgRcptHash, &hash)
	cr.MsgBroker.Send(actor.MsgGasUsed, gas)
	cr.MsgBroker.Send(actor.MsgBloom, bloom)
	cr.CheckPoint("rcpthash calculate completed", zap.Uint64("gas", gas))
	return nil
}

func (cr *CalculateRoothash) gatherReceipts(inclusiveList *types.InclusiveList, receipts []*evmTypes.Receipt) (evmCommon.Hash, evmTypes.Bloom, uint64) {
	var gasused uint64 = 0
	nilroot := evmTypes.EmptyReceiptsHash
	bloom := evmTypes.Bloom{}
	if inclusiveList == nil || receipts == nil {
		return nilroot, bloom, 0
	}

	receiptslist := map[evmCommon.Hash]*evmTypes.Receipt{}
	for _, recp := range receipts {
		receiptslist[recp.TxHash] = recp
	}

	selectedReceipts := make([]*evmTypes.Receipt, 0, len(receipts))
	for i, hash := range inclusiveList.HashList {
		if inclusiveList.Successful[i] {
			if rcpt, ok := receiptslist[*hash]; ok {
				if rcpt != nil {
					selectedReceipts = append(selectedReceipts, rcpt)
					gasused += rcpt.GasUsed
				}
			}
		}
	}
	receiptHash := evmTypes.EmptyReceiptsHash

	if len(selectedReceipts) > 0 {
		receiptHash = evmTypes.DeriveSha(evmTypes.Receipts(selectedReceipts), trie.NewStackTrie(nil))
		bloom = evmTypes.CreateBloom(receipts)
	}
	return receiptHash, bloom, gasused
}
