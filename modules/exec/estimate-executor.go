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
	"math"
	"math/big"

	exetyp "github.com/arcology-network/main/modules/exec/types"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	evmCommon "github.com/ethereum/go-ethereum/common"

	eupk "github.com/arcology-network/eu/common"
	cache "github.com/arcology-network/storage-committer/storage/cache"

	"github.com/arcology-network/common-lib/exp/mempool"

	apihandler "github.com/arcology-network/eu/apihandler"
	eucommon "github.com/arcology-network/eu/common"
	mtypes "github.com/arcology-network/main/types"

	statestore "github.com/arcology-network/storage-committer"
	evmCore "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
)

const (
	estmigateExecStateInit = iota
	estmigateExecStateReady
)

type EstimateExecutor struct {
	state  int
	height uint64

	execParams *exetyp.ExecutorParameter

	taskCh chan *exetyp.ExecMessagers

	chainId *big.Int

	store *statestore.StateStore

	timestamp *big.Int

	euCount int
}

func NewEstimateExecutor() actor.Business {
	exec := &EstimateExecutor{
		state:  estmigateExecStateInit,
		height: math.MaxUint64,
	}

	return exec

}

func (exec *EstimateExecutor) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgApcHandle,
		actor.CombinedName(scommon.MsgBlockStart, scommon.MsgParentInfo),
		scommon.MsgInitialization,
	}, false
}

func (exec *EstimateExecutor) Outputs() map[string]int {
	return map[string]int{}
}

func (exec *EstimateExecutor) Config(params map[string]interface{}) {
	exec.chainId = params["chain_id"].(*big.Int)
	exec.euCount = params["eus"].(int)
}
func (exec *EstimateExecutor) RpcConfig() (string, int) {
	return "estimate-executor", 20
}

func (exec *EstimateExecutor) GetFSMRules() map[int]actor.FSMRule {
	return map[int]actor.FSMRule{
		estmigateExecStateReady: {Accept: []string{
			actor.CombinedName(scommon.MsgBlockStart, scommon.MsgParentInfo),
			scommon.MsgApcHandle,
		}},
		estmigateExecStateInit: {Accept: []string{scommon.MsgInitialization}},
	}
}
func (exec *EstimateExecutor) GetCurrentState() int {
	return exec.state
}

func (exec *EstimateExecutor) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgInitialization, exec.stateInit)
	reg.Register(actor.CombinedName(scommon.MsgBlockStart, scommon.MsgParentInfo), exec.stateReady)
	reg.Register(scommon.MsgApcHandle, exec.updateApc)
	reg.Register("ExecTxs", exec.ExecTxs)
	reg.Register("ExecTxsWithTrace", exec.ExecTxsWithTrace)
}
func (exec *EstimateExecutor) stateInit(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
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
	ctx.ExecCtx.LogDebug("state change into estmigateExecStateReady")
	return nil
}
func (exec *EstimateExecutor) stateReady(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	combined := msg.Data.(*actor.CombinerElements)
	coinbase := evmCommon.BytesToAddress(combined.Get(scommon.MsgBlockStart).Data.(*actor.BlockStart).Coinbase.Bytes())
	exec.height = msg.Height
	exec.execParams = &exetyp.ExecutorParameter{
		ParentInfo: combined.Get(scommon.MsgParentInfo).Data.(*mtypes.ParentInfo),
		Coinbase:   &coinbase,
		Height:     exec.height,
	}
	exec.timestamp = combined.Get(scommon.MsgBlockStart).Data.(*actor.BlockStart).Timestamp
	return nil
}
func (exec *EstimateExecutor) updateApc(ctx *actor.ActionContext) error {
	exec.store = ctx.Messages[0].Data.(*statestore.StateStore)
	return nil
}
func (exec *EstimateExecutor) ExecTxs(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*mtypes.ExecutorRequest)
	task, _, err := exec.newTask(request.Sequences[0])
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
		return nil
	}

	results := exec.execute(task)

	ctx.ExecCtx.SendRpcResponse("", &evmCore.ExecutionResult{
		UsedGas:    results.UsedGas,
		ReturnData: results.ReturnData,
		Err:        results.Err,
	})
	return nil
}
func (exec *EstimateExecutor) ExecTxsWithTrace(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*mtypes.ExecutorRequest)
	for i := range request.Sequences[0].Msgs {
		request.Sequences[0].Msgs[i].Native.SkipAccountChecks = true
	}
	task, tracer, err := exec.newTask(request.Sequences[0])
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
		return err
	}
	exec.execute(task)

	if request.Sequences[0].Config != nil {
		result, err := tracer.GetResult()
		ctx.ExecCtx.SendRpcResponse("", &mtypes.QueryResult{
			Data: result,
		})
		return err
	}
	return nil
}

func (exec *EstimateExecutor) newTask(
	sequence *mtypes.ExecutingSequence,
) (*exetyp.ExecMessagers, tracers.Tracer, error) {
	config := exetyp.MainConfig(exec.chainId)
	config.Coinbase = exec.execParams.Coinbase
	config.BlockNumber = new(big.Int).SetUint64(exec.height)
	config.Time = exec.timestamp
	config.ParentHash = evmCommon.BytesToHash(exec.execParams.ParentInfo.ParentHash.Bytes())
	var tracer tracers.Tracer
	if sequence.Config != nil {

		var err error
		tracer = logger.NewStructLogger(sequence.Config.Config)
		if sequence.Config.Tracer != nil {
			tracer, err = tracers.DefaultDirectory.New(*sequence.Config.Tracer, sequence.Ctx, sequence.Config.TracerConfig)
			if err != nil {
				return nil, nil, err
			}
		}
		config.VMConfig.Tracer = tracer
		config.VMConfig.NoBaseFee = true
	}
	task := &exetyp.ExecMessagers{
		Sequence: sequence,
		Config:   config,
	}

	return task, tracer, nil
}

func (exec *EstimateExecutor) execute(task *exetyp.ExecMessagers) *evmCore.ExecutionResult {
	if task.Sequence.Parallel {
		results := make([]*eucommon.Result, 0, len(task.Sequence.Msgs))

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
			results[0].EvmResult.UsedGas += jobsequence.Jobs[0].PrepaidGas
		}

		return results[0].EvmResult
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

		return jobsequence.Jobs[0].Results.EvmResult
	}
}
