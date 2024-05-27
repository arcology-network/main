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
	"context"
	"sync"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

type RpcService struct {
	actor.WorkerThread

	pendingTxs      map[uint64]chan []*ExecutorResponse
	pendingTxsGuard sync.Mutex
}

var (
	rpcInstance actor.IWorkerEx
	initRpcOnce sync.Once
)

func NewRpcService(concurrency int, groupId string) actor.IWorkerEx {
	initRpcOnce.Do(func() {
		rpc := &RpcService{
			pendingTxs: make(map[uint64]chan []*ExecutorResponse),
		}
		rpc.Set(concurrency, groupId)

		rpcInstance = rpc
	})
	return rpcInstance
}

func (rpc *RpcService) Inputs() ([]string, bool) {
	return []string{
		actor.MsgTxsExecuteResults,
	}, false
}

func (rpc *RpcService) Outputs() map[string]int {
	return map[string]int{
		actor.MsgTxsToExecute: 1,
	}
}

func (rpc *RpcService) OnStart() {}

func (rpc *RpcService) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	rpc.pendingTxsGuard.Lock()
	defer rpc.pendingTxsGuard.Unlock()

	if ch, ok := rpc.pendingTxs[msg.Msgid]; ok {
		ch <- msg.Data.([]*ExecutorResponse)
		delete(rpc.pendingTxs, msg.Msgid)
	} else {
		panic("unexpected msg got")
	}
	return nil
}

func (rpc *RpcService) GetConfig(ctx context.Context, _ *int, config *mtypes.ExecutorConfig) error {
	config.Concurrency = rpc.Concurrency
	return nil
}

func (rpc *RpcService) ExecTxs(ctx context.Context, request *actor.Message, response *mtypes.ExecutorResponses) error {
	chResults := make(chan []*ExecutorResponse)
	rpc.pendingTxsGuard.Lock()
	rpc.pendingTxs[request.Msgid] = chResults
	rpc.pendingTxsGuard.Unlock()

	rpc.MsgBroker.Send(actor.MsgTxsToExecute, request.Data, request.Height, request.Msgid)
	results := <-chResults

	// The following code were copied from exec v1.
	args := request.Data.(*mtypes.ExecutorRequest)
	total := 0
	for _, sequence := range args.Sequences {
		total = total + len(sequence.Msgs)
	}

	resultLength := 0

	HashList := make([]evmCommon.Hash, 0, total)
	StatusList := make([]uint64, 0, total)
	GasUsedList := make([]uint64, 0, total)
	contractAddress := []evmCommon.Address{}

	callResults := make([][]byte, 0, total)

	for _, exectorResponse := range results {

		contractAddress = append(contractAddress, exectorResponse.ContractAddress...)

		resultLength = resultLength + len(exectorResponse.Responses)
		for _, txResponse := range exectorResponse.Responses {

			HashList = append(HashList, txResponse.Hash)
			StatusList = append(StatusList, txResponse.Status)
			GasUsedList = append(GasUsedList, txResponse.GasUsed)
		}

		callResults = append(callResults, exectorResponse.CallResults...)
	}

	rpc.AddLog(log.LogLevel_Debug, "Exec return results***********", zap.Int("txResults", resultLength))

	response.HashList = HashList
	response.StatusList = StatusList
	response.GasUsedList = GasUsedList

	response.ContractAddresses = contractAddress

	response.CallResults = callResults
	return nil
}
