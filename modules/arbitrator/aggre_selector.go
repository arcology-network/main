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

package arbitrator

import (
	ctypes "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/main/modules/arbitrator/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/aggregator/aggregator"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type EuResultsAggreSelector struct {
	ag    *aggregator.Aggregator
	reqId string
	ctx   *actor.ExecutionContext
}

// return a Subscriber struct
func NewEuResultsAggreSelector() actor.Business {
	agg := EuResultsAggreSelector{}
	agg.ag = aggregator.NewAggregator()
	return &agg
}

func (a *EuResultsAggreSelector) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgBlockCompleted,
		scommon.MsgArbitrateReapinglist,
		scommon.MsgPreProcessedEuResults,
	}, false
}

func (a *EuResultsAggreSelector) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgEuResultSelected: 1,
	}
}

func (a *EuResultsAggreSelector) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgBlockCompleted, a.ReceivedBlockCompleted)
	reg.Register(scommon.MsgArbitrateReapinglist, a.ReceivedArbitrateReapinglist)
	reg.Register(scommon.MsgPreProcessedEuResults, a.receivedPreProcessedEuResults)
}

func (a *EuResultsAggreSelector) receivedPreProcessedEuResults(ctx *actor.ActionContext) error {
	data := ctx.Messages[0].Data.([]*types.AccessRecord)
	if len(data) > 0 {
		for _, v := range data {
			euResult := v
			result := a.ag.OnDataReceived(evmCommon.BytesToHash(euResult.TxHash[:]), euResult)
			a.SendMsg(result)
		}
	}
	return nil
}

func (a *EuResultsAggreSelector) ReceivedBlockCompleted(ctx *actor.ActionContext) error {
	remainingQuantity := a.ag.OnClearInfoReceived()
	types.RecordPool.ReclaimRecursive()
	ctx.ExecCtx.LogInfo("clear pool", logger.F("remainingQuantity", remainingQuantity))
	return nil
}
func (a *EuResultsAggreSelector) ReceivedArbitrateReapinglist(ctx *actor.ActionContext) error {
	reapinglist := ctx.Messages[0].Data.(*ctypes.ReapingList)
	// a.reqId = ctx.ExecCtx.GetReqID()
	a.ctx = ctx.ExecCtx.Fork()
	result, _ := a.ag.OnListReceived(reapinglist)
	a.SendMsg(result)
	return nil
}

func (a *EuResultsAggreSelector) ReceivedData(ctx *actor.ActionContext) error {
	data := ctx.Messages[0].Data.([]*types.AccessRecord)

	if len(data) > 0 {
		for _, v := range data {
			euResult := v
			result := a.ag.OnDataReceived(evmCommon.BytesToHash(euResult.TxHash[:]), euResult)
			a.SendMsg(result)
		}
	}
	return nil
}

func (a *EuResultsAggreSelector) SendMsg(selectedData *[]*interface{}) {
	if selectedData != nil {
		euResults := make([]*types.AccessRecord, len(*selectedData))
		for i, euResult := range *selectedData {
			euResults[i] = (*euResult).(*types.AccessRecord)
		}
		a.ctx.LogInfo("send gather result", logger.F("counts", len(euResults)))
		// ctx.ExecCtx.AddRpcReqID(a.reqId)
		a.ctx.Send(scommon.MsgEuResultSelected, euResults)
	}
}
