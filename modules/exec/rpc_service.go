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

package exec

import (
	"sync"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type RpcService struct {
	pendingTxs      map[uint64]chan []*ExecutorResponse
	pendingTxsGuard sync.Mutex
	totalGroups     int
}

func NewRpcService() actor.Business {
	rpc := &RpcService{
		pendingTxs: make(map[uint64]chan []*ExecutorResponse),
	}
	return rpc
}

func (rpc *RpcService) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgTxsExecuteResults,
	}, false
}

func (rpc *RpcService) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgTxsToExecute: 1,
	}
}

func (rpc *RpcService) RpcConfig() (string, int) {
	return "executor", 20
}

func (rs *RpcService) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("startExecute", rs.startExecute)
	reg.Register("GetConfig", rs.GetConfig)
	reg.Register(scommon.MsgTxsExecuteResults, rs.receivedExecuteResults)
}

func (rs *RpcService) startExecute(ctx *actor.ActionContext) error {
	// ctx.ExecCtx.LogDebug("startExecute", logger.F("data", ctx.Messages[0].Data), logger.F("reqID", ctx.Messages[0].ReqID))

	params := ctx.RPC.Request.(*mtypes.ExecutorRequest)
	total := 0
	for _, sequence := range params.Sequences {
		total = total + len(sequence.Msgs)
	}
	rs.totalGroups = total
	ctx.ExecCtx.Send(scommon.MsgTxsToExecute, params, params.Height)
	return nil
}

func (rs *RpcService) receivedExecuteResults(ctx *actor.ActionContext) error {
	resp := ctx.Messages[0].Data.([]*ExecutorResponse)
	resultLength := 0

	HashList := make([]evmCommon.Hash, 0, rs.totalGroups)
	StatusList := make([]uint64, 0, rs.totalGroups)
	GasUsedList := make([]uint64, 0, rs.totalGroups)
	contractAddress := []evmCommon.Address{}

	callResults := make([][]byte, 0, rs.totalGroups)

	for _, exectorResponse := range resp {

		contractAddress = append(contractAddress, exectorResponse.ContractAddress...)

		resultLength = resultLength + len(exectorResponse.Responses)
		for _, txResponse := range exectorResponse.Responses {

			HashList = append(HashList, txResponse.Hash)
			StatusList = append(StatusList, txResponse.Status)
			GasUsedList = append(GasUsedList, txResponse.GasUsed)
		}

		callResults = append(callResults, exectorResponse.CallResults...)
	}

	ctx.ExecCtx.LogDebug("Exec return results", logger.F("txResults", resultLength))

	ctx.ExecCtx.SendRpcResponse("", &mtypes.ExecutorResponses{
		HashList:          HashList,
		StatusList:        StatusList,
		GasUsedList:       GasUsedList,
		ContractAddresses: contractAddress,
		CallResults:       callResults,
	})

	return nil
}

func (rs *RpcService) GetConfig(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", &mtypes.ExecutorConfig{Concurrency: ctx.ExecCtx.Concurrency()})
	return nil
}
