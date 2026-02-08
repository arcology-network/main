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

package ethapi

import (
	"math/big"
	"time"

	"github.com/arcology-network/common-lib/common"
	internal "github.com/arcology-network/main/modules/eth-api/backend"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	evmCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

type FilterManager struct {
	filters *internal.Filters
	sender  actor.OutboundSender
}

// return a Subscriber struct
func NewFilterManager() actor.Business {
	fm := FilterManager{}
	return &fm
}
func (fm *FilterManager) SetSender(sender actor.OutboundSender) {
	fm.sender = sender

}
func (fm *FilterManager) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgSelectedReceipts,
		scommon.MsgPendingBlock,
	}, true
}
func (fm *FilterManager) PrimaryMsg() string {
	return scommon.MsgSelectedReceipts
}
func (fm *FilterManager) Outputs() map[string]int {
	return map[string]int{}
}

func (fm *FilterManager) Config(params map[string]interface{}) {
	fm.filters = internal.NewFilters()
	fm.filters.SetTimeout(time.Minute * time.Duration(params["filter_timeout_mins"].(int)))

	options.KeyFile = params["key_file"].(string)
	options.Port = uint64(params["json_rpc_port"].(int))
	options.AuthPort = uint64(params["auth_rpc_port"].(int))
	options.Debug = params["debug"].(bool)
	options.Waits = params["retry_time"].(int)
	options.ProtocolVersion = params["protocol_version"].(int)
	options.Hashrate = params["hash_rate"].(int)
	options.ChainID = params["chain_id"].(*big.Int).Uint64()
	options.JwtFile = params["jwt_file"].(string)

	startJsonRpc(fm.sender)
	startAuthJsonRpc()
}
func (fm *FilterManager) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgSelectedReceipts, fm.SetData)
	reg.Register(scommon.MsgPendingBlock, fm.SetData)
}

func (fm *FilterManager) SetData(ctx *actor.ActionContext) error {
	var receipts []*ethTypes.Receipt
	var block *mtypes.MonacoBlock

	for _, v := range ctx.Messages {
		switch v.Name {
		case scommon.MsgSelectedReceipts:
			receipts = v.Data.([]*ethTypes.Receipt)
		case scommon.MsgPendingBlock:
			block = v.Data.(*mtypes.MonacoBlock)
		}
	}

	blockHash := block.Hash()
	worker := func(start, end int, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			receipts[i].BlockHash = evmCommon.BytesToHash(blockHash)
			receipts[i].BlockNumber = big.NewInt(int64(block.Height))
			receipts[i].TransactionIndex = uint(i)

			for k := range receipts[i].Logs {
				receipts[i].Logs[k].BlockHash = receipts[i].BlockHash
				receipts[i].Logs[k].TxHash = receipts[i].TxHash
				receipts[i].Logs[k].TxIndex = receipts[i].TransactionIndex
			}
		}
	}

	common.ParallelWorker(len(receipts), ctx.ExecCtx.Concurrency(), worker)
	fm.filters.OnResultsArrived(block.Height, receipts, evmCommon.BytesToHash(blockHash))

	return nil
}
