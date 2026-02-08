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
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

type CalculateRoothash struct {
}

// return a Subscriber struct
func NewCalculateRoothash() actor.Business {
	return &CalculateRoothash{}
}

func (cr *CalculateRoothash) Inputs() ([]string, bool) {
	return []string{scommon.MsgSelectedReceipts, scommon.MsgInclusive}, true
}

func (cr *CalculateRoothash) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgReceiptInfo: 1,
	}
}

func (cr *CalculateRoothash) PrimaryMsg() string {
	return scommon.MsgInclusive
}

func (cr *CalculateRoothash) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgSelectedReceipts, cr.startCalculateHash)
	reg.Register(scommon.MsgInclusive, cr.startCalculateHash)
}

func (cr *CalculateRoothash) startCalculateHash(ctx *actor.ActionContext) error {
	var inclusiveList *types.InclusiveList
	var selectedReceipts []*evmTypes.Receipt
	for _, v := range ctx.Messages {
		switch v.Name {
		case scommon.MsgInclusive:
			inclusiveList = v.Data.(*types.InclusiveList)
		case scommon.MsgSelectedReceipts:
			selectedReceipts = v.Data.([]*evmTypes.Receipt)
		}
	}
	ctx.ExecCtx.LogInfo("start calculate rcpthash", logger.F("inclusiveList", len(inclusiveList.HashList)), logger.F("selectedReceipts", len(selectedReceipts)))
	hash, bloom, gas, successfulTxs := cr.gatherReceipts(inclusiveList, selectedReceipts)
	ctx.ExecCtx.Send(scommon.MsgReceiptInfo, &mtypes.ReceiptInfo{
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
	ctx.ExecCtx.LogInfo("rcpthash calculate completed", logger.F("gas", gas))
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
