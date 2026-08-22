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
	"github.com/arcology-network/common-lib/exp/slice"
	ctypes "github.com/arcology-network/common-lib/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"

	"github.com/arcology-network/common-lib/crdt/statecell"
	"github.com/arcology-network/scheduler/conflictor"
)

type RpcService struct {
	arbitrator *conflictor.Conflictor
}

func NewRpcService() actor.Business {
	rs := RpcService{}
	rs.arbitrator = conflictor.NewConflictor()
	return &rs
}

func (rs *RpcService) RpcConfig() (string, int) {
	return "arbitrator", 20
}

func (rs *RpcService) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgAccessRecordSelected,
		// scommon.MsgPreProcessedImportEuResults,
	}, false
}

func (rs *RpcService) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgArbitrateReapinglist:  1,
		scommon.MsgPreProcessedEuResults: 1,
	}
}

func (rs *RpcService) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("startArbitrate", rs.startArbitrate)
	reg.Register(scommon.MsgAccessRecordSelected, rs.receivedEuResultSelected)
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
	resultSelected := msg.Data.([]*statecell.StateCell)

	if resultSelected != nil {
		ctx.ExecCtx.LogInfo("Before detectConflict", logger.F("tx nums", len(resultSelected)))

		rs.arbitrator.Insert(resultSelected)
		collision, _, _ := rs.arbitrator.Detect()
		summery := conflictor.NewCollisionSummary(resultSelected, collision)

		ctx.ExecCtx.SendRpcResponse("", summery)
		ctx.ExecCtx.LogInfo("arbitrate return results", logger.F("Collisions", len(summery.Collisions)))
	}
	rs.arbitrator.Clear()

	return nil
}
