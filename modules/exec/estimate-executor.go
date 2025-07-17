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
	"math"
	"math/big"
	"sync"

	"github.com/arcology-network/main/components/storage"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"

	eupk "github.com/arcology-network/eu/common"
	cache "github.com/arcology-network/storage-committer/storage/cache"
	univaluepk "github.com/arcology-network/storage-committer/type/univalue"

	"github.com/arcology-network/common-lib/exp/mempool"

	apihandler "github.com/arcology-network/eu/apihandler"
	eucommon "github.com/arcology-network/eu/common"
	mtypes "github.com/arcology-network/main/types"

	cmncmn "github.com/arcology-network/common-lib/common"
	statestore "github.com/arcology-network/storage-committer"
	evmCore "github.com/ethereum/go-ethereum/core"
)

const (
	estmigateExecStateInit = iota
	estmigateExecStateReady
)

var (
	execInstance actor.IWorkerEx
	initExecOnce sync.Once
)

type EstimateExecutor struct {
	actor.WorkerThread

	state  int
	height uint64

	execParams *exetyp.ExecutorParameter

	taskCh chan *exetyp.ExecMessagers

	chainId *big.Int

	store *statestore.StateStore

	timestamp *big.Int

	pendingTxs      map[uint64]chan *evmCore.ExecutionResult
	pendingTxsGuard sync.Mutex
}

func NewEstimateExecutor(concurrency int, groupId string) actor.IWorkerEx {
	initExecOnce.Do(func() {
		exec := &EstimateExecutor{
			state:      estmigateExecStateInit,
			height:     math.MaxUint64,
			taskCh:     make(chan *exetyp.ExecMessagers, concurrency),
			pendingTxs: make(map[uint64]chan *evmCore.ExecutionResult),
		}
		exec.Set(concurrency, groupId)
		execInstance = exec
	})
	return execInstance
}

func (exec *EstimateExecutor) Inputs() ([]string, bool) {
	return []string{
		actor.MsgApcHandle,
		actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo),
		actor.MsgInitialization,
	}, false
}

func (exec *EstimateExecutor) Outputs() map[string]int {
	return map[string]int{}
}

func (exec *EstimateExecutor) Config(params map[string]interface{}) {
	exec.chainId = params["chain_id"].(*big.Int)
}

func (exec *EstimateExecutor) OnStart() {
	exec.startExec()
}

func (exec *EstimateExecutor) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch exec.state {
	case estmigateExecStateInit:
		initialization := msg.Data.(*mtypes.Initialization)
		exec.store = initialization.Store
		exec.height = msg.Height

		addr := evmCommon.BytesToAddress(initialization.BlockStart.Coinbase.Bytes())
		exec.execParams = &exetyp.ExecutorParameter{
			ParentInfo: initialization.ParentInformation,
			Coinbase:   &addr,
			Height:     exec.height,
		}
		exec.timestamp = initialization.BlockStart.Timestamp
		exec.state = estmigateExecStateReady
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>state change into estmigateExecStateReady")
	case estmigateExecStateReady:
		switch msg.Name {
		case actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo):
			combined := msg.Data.(*actor.CombinerElements)
			coinbase := evmCommon.BytesToAddress(combined.Get(actor.MsgBlockStart).Data.(*actor.BlockStart).Coinbase.Bytes())
			exec.height = msg.Height
			exec.execParams = &exetyp.ExecutorParameter{
				ParentInfo: combined.Get(actor.MsgParentInfo).Data.(*mtypes.ParentInfo),
				Coinbase:   &coinbase,
				Height:     exec.height,
			}
			exec.timestamp = combined.Get(actor.MsgBlockStart).Data.(*actor.BlockStart).Timestamp
		case actor.MsgApcHandle:
			exec.store = msg.Data.(*statestore.StateStore)
		}
	}
	return nil
}

func (exec *EstimateExecutor) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		estmigateExecStateReady: {
			actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo),
			actor.MsgApcHandle,
		},
		estmigateExecStateInit: {
			actor.MsgInitialization,
		},
	}
}

func (exec *EstimateExecutor) GetCurrentState() int {
	return exec.state
}

func (exec *EstimateExecutor) ExecTxs(ctx context.Context, request *mtypes.ExecutorRequest, response *evmCore.ExecutionResult) error {
	chResults := make(chan *evmCore.ExecutionResult)
	exec.pendingTxsGuard.Lock()
	msgId := cmncmn.GenerateUUID()
	exec.pendingTxs[msgId] = chResults
	exec.pendingTxsGuard.Unlock()

	exec.sendNewTask(request.Sequences[0], msgId)
	results := <-chResults

	response.UsedGas = results.UsedGas
	response.ReturnData = results.ReturnData
	response.Err = results.Err
	return nil
}

func (exec *EstimateExecutor) sendNewTask(
	sequence *mtypes.ExecutingSequence,
	msgid uint64,
) {
	config := exetyp.MainConfig(exec.chainId)
	config.Coinbase = exec.execParams.Coinbase
	config.BlockNumber = new(big.Int).SetUint64(exec.height)
	config.Time = exec.timestamp
	config.ParentHash = evmCommon.BytesToHash(exec.execParams.ParentInfo.ParentHash.Bytes())
	task := &exetyp.ExecMessagers{
		Sequence: sequence,
		Config:   config,
		Debug:    true,
		Msgid:    msgid,
	}
	exec.taskCh <- task
}

func (exec *EstimateExecutor) execute(task *exetyp.ExecMessagers) {
	storage.RequestLock("exec", "estimateExecutor")
	defer storage.ReleaseLock("exec", "estimateExecutor")

	if task.Sequence.Parallel {
		results := make([]*eucommon.Result, 0, len(task.Sequence.Msgs))
		mtransitions := make(map[uint64][]*univaluepk.Univalue, len(task.Sequence.Msgs))
		for j := range task.Sequence.Msgs {

			api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
				return exec.store.WriteCache
			}, func(cache *cache.WriteCache) { cache.Clear() }))
			jobsequence := eupk.JobSequence{
				ID:     uint64(j),
				SeqAPI: api,
			}
			jobsequence.AppendMsg(task.Sequence.Msgs[j])

			jobsequence.Run(task.Config, api, GetThreadID(jobsequence.Jobs[0].StdMsg.TxHash))
			results = append(results, jobsequence.Jobs[0].Results)
			mtransitions[uint64(task.Sequence.Msgs[j].ID)] = jobsequence.Jobs[0].Results.Transitions()
		}
		exec.sendResults(task.Msgid, results[0].EvmResult)
	} else {
		api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
			return exec.store.WriteCache
		}, func(cache *cache.WriteCache) { cache.Clear() }))

		jobsequence := eupk.JobSequence{
			ID:     uint64(0),
			SeqAPI: api,
		}
		for i := range task.Sequence.Msgs {
			jobsequence.AppendMsg(task.Sequence.Msgs[i])
		}

		jobsequence.Run(task.Config, api, GetThreadID(task.Sequence.Msgs[0].TxHash))
		results := make([]*eucommon.Result, len(task.Sequence.Msgs))
		for i := range task.Sequence.Msgs {
			results[i] = jobsequence.Jobs[i].Results
		}
		exec.sendResults(task.Msgid, jobsequence.Jobs[0].Results.EvmResult)
	}
}

func (exec *EstimateExecutor) startExec() {
	go func() {
		for {
			task := <-exec.taskCh
			exec.execute(task)
		}
	}()
}

func (exec *EstimateExecutor) sendResults(msgid uint64, results *evmCore.ExecutionResult) {
	exec.pendingTxsGuard.Lock()
	defer exec.pendingTxsGuard.Unlock()

	if ch, ok := exec.pendingTxs[msgid]; ok {
		ch <- results
		delete(exec.pendingTxs, msgid)
	} else {
		panic("unexpected result got")
	}
}
