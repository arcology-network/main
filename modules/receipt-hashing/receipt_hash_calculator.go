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

package receipthashing

import (
	"time"

	"github.com/arcology-network/common-lib/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"go.uber.org/zap"
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
		actor.MsgReceiptInfo: 1,
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
			// for _, item := range v.Data.([]interface{}) {
			// 	selectedReceipts = append(selectedReceipts, item.(*evmTypes.Receipt))
			// }
			selectedReceipts = v.Data.([]*evmTypes.Receipt)
		}
	}
	cr.CheckPoint("start calculate rcpthash")
	hash, bloom, gas, successfulTxs := cr.gatherReceipts(inclusiveList, selectedReceipts)
	cr.MsgBroker.Send(actor.MsgReceiptInfo, &mtypes.ReceiptInfo{
		RcptHash:  hash,
		BloomInfo: bloom,
		Gasused:   gas,
		TpsGas: &mtypes.TPSGasBurned{
			TotalTxs:      uint64(len(inclusiveList.HashList)),
			SuccessfulTxs: uint64(successfulTxs),
			GasUsed:       gas,
			Timestamp:     time.Now().UnixMilli(),
		},
	})
	cr.CheckPoint("rcpthash calculate completed", zap.Uint64("gas", gas))
	return nil
}

func (cr *CalculateRoothash) gatherReceipts(inclusiveList *types.InclusiveList, receipts []*evmTypes.Receipt) (evmCommon.Hash, evmTypes.Bloom, uint64, int) {
	var gasused uint64 = 0
	nilroot := evmTypes.EmptyReceiptsHash
	bloom := evmTypes.Bloom{}
	if inclusiveList == nil || receipts == nil {
		return nilroot, bloom, 0, 0
	}

	receiptslist := map[evmCommon.Hash]*evmTypes.Receipt{}
	for _, recp := range receipts {
		receiptslist[recp.TxHash] = recp
	}

	successfulTxs := 0
	selectedReceipts := make([]*evmTypes.Receipt, 0, len(receipts))
	for i, hash := range inclusiveList.HashList {
		if inclusiveList.Successful[i] {
			successfulTxs = successfulTxs + 1
			if rcpt, ok := receiptslist[hash]; ok {
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
	return receiptHash, bloom, gasused, successfulTxs
}
