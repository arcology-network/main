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
	"errors"
	"math"
	"math/big"

	exetyp "github.com/arcology-network/main/modules/exec/types"
	"github.com/arcology-network/scheduler/workload"
	statecache "github.com/arcology-network/state-engine/state/cache"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	evmCommon "github.com/ethereum/go-ethereum/common"

	"github.com/arcology-network/eu/eu"

	"github.com/arcology-network/common-lib/exp/mempool"

	apihandler "github.com/arcology-network/eu/apihandler"
	mtypes "github.com/arcology-network/main/types"

	eucommon "github.com/arcology-network/common-lib/types"
	proxy "github.com/arcology-network/state-engine/storage/proxy"
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

	store *statecache.ExecutionStateStore

	timestamp *big.Int

	euCount int

	requestStore map[string]*mtypes.ExecutorDebugRequest
}

func NewEstimateExecutor() actor.Business {
	exec := &EstimateExecutor{
		state:        estmigateExecStateInit,
		height:       math.MaxUint64,
		requestStore: map[string]*mtypes.ExecutorDebugRequest{},
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
	reg.Register("DebugExecTxs", exec.DebugExecTxs)
	reg.Register("onStateRoot", exec.onStateRoot)
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
	exec.store = ctx.Messages[0].Data.(*statecache.ExecutionStateStore)
	return nil
}

func (exec *EstimateExecutor) DebugExecTxs(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*mtypes.ExecutorDebugRequest)
	reqId := ctx.ExecCtx.Current.ReqID
	exec.requestStore[reqId] = request

	if request.BlockParams != nil {
		ctx.ExecCtx.InvokeRPC("storage", "Query", &mtypes.QueryRequest{
			QueryType: mtypes.QueryType_State_Root,
			Data: &mtypes.StateRootRequest{
				BlockParam: request.BlockParams,
				ReqId:      reqId,
			},
		}, "onStateRoot")
	} else {
		exec.startExec(ctx, reqId)
	}
	return nil
}

func (exec *EstimateExecutor) startExec(ctx *actor.ActionContext, reqId string) error {
	exec.process(ctx, reqId, exec.store)
	return nil
}

func (exec *EstimateExecutor) onStateRoot(ctx *actor.ActionContext) error {
	resp := ctx.RPC.Request.(*mtypes.QueryResult).Data.(*mtypes.StateRootResponse)
	tri, err := proxy.NewEthStateSnapshot([32]byte(resp.Root), exec.store.CommittedStore().(*proxy.StorageProxy).EthStore().TrieDB())
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
		return err
	}
	store := statecache.NewDefaultExecutionStateStore(tri)

	exec.process(ctx, resp.ReqId, store)
	return nil
}

func (exec *EstimateExecutor) process(ctx *actor.ActionContext, reqId string, store *statecache.ExecutionStateStore) error {
	if request, ok := exec.requestStore[reqId]; ok {
		delete(exec.requestStore, reqId)

		task, tracer, err := exec.newTask(request, request.Msg)
		if err != nil {
			ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
			return err
		}

		results := exec.execute(task, store)

		if request.Config != nil {
			result, err := tracer.GetResult()
			ctx.ExecCtx.SendRpcResponse("", &mtypes.QueryResult{
				Data: result,
			})
			return err
		} else {
			ctx.ExecCtx.SendRpcResponse("", &evmCore.ExecutionResult{
				UsedGas:    results.UsedGas,
				ReturnData: results.ReturnData,
				Err:        results.Err,
			})
		}
	} else {
		errstr := "request not found"
		ctx.ExecCtx.SendRpcResponse(errstr, nil)
		return errors.New(errstr)
	}

	return nil
}

func (exec *EstimateExecutor) newTask(
	request *mtypes.ExecutorDebugRequest,
	stdmsg *eucommon.StandardMessage,
) (*exetyp.ExecMessagers, *tracers.Tracer, error) {
	config := exetyp.MainConfig(exec.chainId)
	config.Coinbase = exec.execParams.Coinbase
	config.BlockNumber = new(big.Int).SetUint64(exec.height)
	config.Time = exec.timestamp
	config.ParentHash = evmCommon.BytesToHash(exec.execParams.ParentInfo.ParentHash.Bytes())
	var tracer *tracers.Tracer
	if request.Config != nil {
		var err error
		structLogger := logger.NewStructLogger(request.Config.Config)
		tracer = &tracers.Tracer{
			Hooks:     structLogger.Hooks(),
			GetResult: structLogger.GetResult,
			Stop:      structLogger.Stop,
		}
		if request.Config.Tracer != nil {
			tracer, err = tracers.DefaultDirectory.New(*request.Config.Tracer, request.Ctx, request.Config.TracerConfig, config.ChainConfig)
			if err != nil {
				return nil, nil, err
			}
		}
		config.VMConfig.Tracer = tracer.Hooks
		config.VMConfig.NoBaseFee = true
	}
	task := &exetyp.ExecMessagers{
		Sequence: workload.NewJobSequenceFromStandardMessages(0, stdmsg),
		Config:   config,
	}

	return task, tracer, nil
}

func (exec *EstimateExecutor) execute(task *exetyp.ExecMessagers, store *statecache.ExecutionStateStore) *evmCore.ExecutionResult {
	pipeline := eu.ExecutionPipeline{
		NumThreads: 1,
		Config:     task.Config,
	}

	ccRuntime := apihandler.NewConcurrentRuntime(
		0, // concurrent runtime ID
		mempool.NewMempool(
			16,
			1,
			func() *statecache.ExecutionStateStore {
				// When creating a new writecache, use store as the backend.
				return statecache.NewExecutionStateStore(store, 32, 1)
			},
			func(cache *statecache.ExecutionStateStore) {
				cache.Clear()
			}),
	)

	pipeline.RunSequence(&workload.Generation{ID: uint64(0)}, task.Sequence, ccRuntime.Cascade(uint64(0)), uint64(0))

	results := make([]*workload.Result, len(task.Sequence.Jobs))
	for i := range task.Sequence.Jobs {
		results[i] = task.Sequence.Jobs[i].Result
	}

	return task.Sequence.Jobs[0].Result.EvmResult
}
