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
	"fmt"

	"github.com/arcology-network/common-lib/exp/slice"
	ctypes "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/main/modules/arbitrator/types"
	mtypes "github.com/arcology-network/main/types"
	arbitratorn "github.com/arcology-network/scheduler/arbitrator"
	"github.com/arcology-network/storage-committer/type/univalue"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type RpcService struct {
	arbitrator *arbitratorn.Arbitrator
}

func NewRpcService() actor.Business {
	rs := RpcService{}
	rs.arbitrator = arbitratorn.NewArbitrator()
	return &rs
}
func (rs *RpcService) RpcConfig() (string, int) {
	return "arbitrator", 20
}
func (rs *RpcService) Inputs() ([]string, bool) {
	return []string{scommon.MsgEuResultSelected, scommon.MsgPreProcessedImportEuResults}, false
}

func (rs *RpcService) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgArbitrateReapinglist:  1,
		scommon.MsgPreProcessedEuResults: 1,
	}
}

func (rs *RpcService) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("startArbitrate", rs.startArbitrate)
	reg.Register(scommon.MsgEuResultSelected, rs.receivedEuResultSelected)
	reg.Register(scommon.MsgPreProcessedImportEuResults, rs.receivedPreProcessedEuResults)
}

func (rs *RpcService) startArbitrate(ctx *actor.ActionContext) error {
	params := ctx.RPC.Request.(*mtypes.ArbitratorRequest)
	reapinglist := ctypes.ReapingList{
		List: slice.Flatten(params.TxsListGroup),
	}

	ctx.ExecCtx.Send(scommon.MsgArbitrateReapinglist, &reapinglist)

	return nil
}

func (rs *RpcService) receivedEuResultSelected(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	resultSelected := msg.Data.([]*types.AccessRecord)

	if resultSelected != nil && len(resultSelected) > 0 {
		ctx.ExecCtx.LogInfo("Before detectConflict", logger.F("tx nums", len(resultSelected)))
		conflicts := rs.arbitrator.Detect()
		fmt.Printf("----------arbitrate result-------conflicts Info------------\n")
		arbitratorn.Conflicts(conflicts).Print()

		left, right := parseResult(conflicts)

		ctx.ExecCtx.SendRpcResponse("", &mtypes.ArbitratorResponse{
			CPairLeft:  left,
			CPairRight: right,
		})
		ctx.ExecCtx.LogInfo("arbitrate return results", logger.F("left", left), logger.F("right", right))
	}
	rs.arbitrator.Clear()

	return nil
}

func (rs *RpcService) receivedPreProcessedEuResults(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ars := msg.Data.([]*types.AccessRecord)
	newTrans := make([]*univalue.Univalue, 0, len(ars)*50)
	for i := range ars {
		newTrans = append(newTrans, ars[i].Accesses...)
	}
	rs.arbitrator.Insert(newTrans)

	ctx.ExecCtx.Send(scommon.MsgPreProcessedEuResults, ars)
	return nil
}

func parseRequests(txsListGroup [][]evmCommon.Hash, results *[]*types.AccessRecord) ([][]uint64, [][]*univalue.Univalue) {
	mp := map[[32]byte]*types.AccessRecord{}
	for _, result := range *results {
		mp[result.TxHash] = result
	}
	groupIDs := make([][]uint64, len(txsListGroup))
	records := make([][]*univalue.Univalue, len(txsListGroup))
	for i, row := range txsListGroup {
		ids := make([]uint64, 0, len(row))
		transactations := []*univalue.Univalue{}
		for _, e := range row {
			result := mp[[32]byte(e.Bytes())]
			ids = append(ids, slice.Fill(make([]uint64, len(result.Accesses)), uint64(i))...)
			transactations = append(transactations, result.Accesses...)
		}
		groupIDs[i] = ids
		records[i] = transactations
	}
	return groupIDs, records
}

func parseResult(conflits arbitratorn.Conflicts) ([]uint64, []uint64) {
	_, _, pairs := conflits.ToDict()
	left := make([]uint64, 0, len(pairs))
	right := make([]uint64, 0, len(pairs))
	for _, pair := range pairs {
		left = append(left, pair[0])
		right = append(right, pair[1])
	}
	return left, right
}
