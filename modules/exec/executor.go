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
	"fmt"
	"math"
	"math/big"
	"runtime"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/types"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"

	"github.com/arcology-network/common-lib/crdt/statecell"
	"github.com/arcology-network/common-lib/exp/mempool"
	"github.com/arcology-network/common-lib/exp/slice"

	apihandler "github.com/arcology-network/eu/apihandler"
	eushared "github.com/arcology-network/eu/shared"
	mtypes "github.com/arcology-network/main/types"
	evmTypes "github.com/ethereum/go-ethereum/core/types"

	scommon "github.com/arcology-network/streamer/common"

	"github.com/arcology-network/eu/eu"
	statecache "github.com/arcology-network/state-engine/state/cache"

	workload "github.com/arcology-network/scheduler/workload"
)

type ExecutorResponse struct {
	Responses       []*mtypes.ExecuteResponse
	ContractAddress []evmCommon.Address
	CallResults     [][]byte
}

const (
	execStateWaitBlockStart = iota
	execStateWaitGenerationReady
	execStateReady
	execStateInit
	execStateNextHeight
)

type Executor struct {
	state         int
	height        uint64
	generationIdx uint32

	// snapshotDict SnapshotDict
	execParams *exetyp.ExecutorParameter

	taskCh   chan *exetyp.ExecMessagers
	resultCh chan *mtypes.JobSequenceResponse
	numTasks int
	ctx      *actor.ExecutionContext

	chainId *big.Int

	store     *statecache.ExecutionStateStore
	stateInit bool

	euCount int

	ExecId uint32
}

func NewExecutor() actor.Business {
	exec := &Executor{
		state:     execStateInit,
		height:    math.MaxUint64,
		stateInit: false,
	}

	return exec
}

func (exec *Executor) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgApcHandle, // Init DB on every generation.
		actor.CombinedName(scommon.MsgBlockStart, scommon.MsgParentInfo, scommon.MsgObjectCached), // Got block context.
		scommon.MsgGenerationReapingList, //update generationIdx
		scommon.MsgBlockEnd,
		scommon.MsgInitialization,
	}, false
}

func (exec *Executor) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgReceipts:        100, // Exec results.
		scommon.MsgEuResults:       100, // Exec results.
		scommon.MsgNonceEuResults:  100,
		scommon.MsgTxAccessRecords: 100, // Access records for arbitrator.
	}
}

func (exec *Executor) Config(params map[string]interface{}) {
	exec.chainId = params["chain_id"].(*big.Int)
	exec.euCount = params["eus"].(int)
	exec.taskCh = make(chan *exetyp.ExecMessagers, mtypes.MaxBlockSize)
	exec.resultCh = make(chan *mtypes.JobSequenceResponse, exec.euCount)
	exec.startExec()
}

func (exec *Executor) RpcConfig() (string, int) {
	return "executor", 20
}

func (exec *Executor) GetFSMRules() map[int]actor.FSMRule {
	return map[int]actor.FSMRule{
		execStateInit: {Accept: []string{
			scommon.MsgInitialization,
		}},
		execStateWaitBlockStart: {Accept: []string{
			actor.CombinedName(scommon.MsgBlockStart, scommon.MsgParentInfo, scommon.MsgObjectCached),
		}},
		execStateWaitGenerationReady: {Accept: []string{
			scommon.MsgApcHandle,
		}},
		execStateReady: {Accept: []string{
			scommon.MsgGenerationReapingList,
		}},
		execStateNextHeight: {Accept: []string{
			scommon.MsgBlockEnd,
		}},
	}
}

func (exec *Executor) GetCurrentState() int {
	return exec.state
}

func (exec *Executor) Height() uint64 {
	return exec.height
}

func (exec *Executor) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(actor.CombinedName(scommon.MsgBlockStart, scommon.MsgParentInfo, scommon.MsgObjectCached), exec.waitBlockStart)
	reg.Register(scommon.MsgApcHandle, exec.waitGenerationReady)
	reg.Register(scommon.MsgInitialization, exec.StateInit)
	reg.Register(scommon.MsgBlockEnd, exec.nextHeight)
	reg.Register(scommon.MsgGenerationReapingList, exec.receivedGenerationReapingList)

	reg.Register("startExecute", exec.startExecute)
}

func (exec *Executor) waitBlockStart(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	combined := msg.Data.(*actor.CombinerElements)
	coinbase := evmCommon.BytesToAddress(combined.Get(scommon.MsgBlockStart).Data.(*actor.BlockStart).Coinbase.Bytes())
	exec.execParams = &exetyp.ExecutorParameter{
		ParentInfo: combined.Get(scommon.MsgParentInfo).Data.(*mtypes.ParentInfo),
		Coinbase:   &coinbase,
		Height:     exec.height,
	}

	exec.ChangeState(ctx, execStateWaitGenerationReady, "execStateWaitGenerationReady")
	return nil
}
func (exec *Executor) waitGenerationReady(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	exec.store = msg.Data.(*statecache.ExecutionStateStore)
	exec.ChangeState(ctx, execStateReady, "execStateReady")
	return nil
}
func (exec *Executor) StateInit(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	initialization := msg.Data.(*mtypes.Initialization)
	exec.store = initialization.Store
	exec.height = msg.Height + 1

	addr := evmCommon.BytesToAddress(initialization.BlockStart.Coinbase.Bytes())
	exec.execParams = &exetyp.ExecutorParameter{
		ParentInfo: initialization.ParentInformation,
		Coinbase:   &addr,
		Height:     exec.height,
	}

	exec.ChangeState(ctx, execStateReady, "execStateReady")
	return nil
}

func (exec *Executor) nextHeight(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	exec.height = msg.Height + 1
	exec.ChangeState(ctx, execStateWaitBlockStart, "execStateWaitBlockStart")
	return nil
}
func (exec *Executor) ChangeState(ctx *actor.ActionContext, state int, stateName string) error {
	ctx.ExecCtx.LogDebug("****** " + ctx.ExecCtx.WorkCtx.BusinassName + " state change into " + stateName)
	exec.state = state
	return nil
}

func (exec *Executor) startExecute(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*mtypes.ExecutorRequest)
	exec.ExecId = request.ExecId

	exec.numTasks = len(request.JobSequences)
	exec.ctx = ctx.ExecCtx.Fork()
	if exec.generationIdx != request.GenerationIdx {
		panic("generationIdx not match!")
	}
	ctx.ExecCtx.LogDebug("newTxsPack sequenct counter", logger.F("counter", len(request.JobSequences)))
	for i := range request.JobSequences {
		exec.sendNewTask(request.Timestamp, request.JobSequences[i])
	}
	exec.collectResults(ctx, len(request.JobSequences))
	return nil
}

func (exec *Executor) collectResults(ctx *actor.ActionContext, taskNums int) {
	responses := make([]*mtypes.JobSequenceResponse, 0, taskNums)

	for {
		responses = append(responses, <-exec.resultCh)
		if len(responses) == taskNums {
			break
		}
	}

	ctx.ExecCtx.SendRpcResponse("", &mtypes.ExecResponses{
		Resp:   responses,
		ExecId: exec.ExecId,
	})

}

func (exec *Executor) receivedGenerationReapingList(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	reapinglist := msg.Data.(*types.InclusiveList)
	exec.generationIdx = reapinglist.NextGenerationIdx
	if reapinglist.NextGenerationIdx > 0 {
		exec.ChangeState(ctx, execStateWaitGenerationReady, "execStateWaitGenerationReady")
	} else {
		exec.ChangeState(ctx, execStateNextHeight, "execStateNextHeight")
	}
	return nil
}

func (exec *Executor) sendNewTask(
	timestamp *big.Int,
	sequence *workload.JobSequence,
) {
	config := exetyp.MainConfig(exec.chainId)
	config.Coinbase = exec.execParams.Coinbase
	config.BlockNumber = new(big.Int).SetUint64(exec.height)
	config.Time = timestamp
	config.ParentHash = evmCommon.BytesToHash(exec.execParams.ParentInfo.ParentHash.Bytes())
	task := &exetyp.ExecMessagers{
		Sequence: sequence,
		Config:   config,
	}
	exec.taskCh <- task
}

func GetThreadID(hash evmCommon.Hash) uint64 {
	return uint64(codec.Uint64(0).Decode(hash.Bytes()[:8]).(codec.Uint64))
}
func (exec *Executor) startExec() {
	for i := 0; i < int(exec.euCount); i++ {
		index := i
		go func(index int) {
			for {
				task := <-exec.taskCh
				exec.ctx.LogDebug("start execute", logger.F("txs counter", len(task.Sequence.Jobs)))

				pipeline := eu.ExecutionPipeline{
					NumThreads: uint32(exec.euCount),
					Config:     task.Config,
				}
				ccRuntime := apihandler.NewConcurrentRuntime(
					0, // concurrent runtime ID
					mempool.NewMempool(
						16,
						1,
						func() *statecache.ExecutionStateStore {
							// When creating a new writecache, use store as the backend.
							return statecache.NewExecutionStateStore(exec.store, 32, 1)
						},
						func(cache *statecache.ExecutionStateStore) {
							cache.Clear()
						}),
				)

				transitions, _ := pipeline.RunSequence(&workload.Generation{ID: uint64(exec.generationIdx)}, task.Sequence, ccRuntime.Cascade(uint64(0)), uint64(0))

				mtransitions := exec.parseResults(transitions)

				results := make([]*workload.Result, len(task.Sequence.Jobs))
				for i := range task.Sequence.Jobs {
					results[i] = task.Sequence.Jobs[i].Result
				}

				exec.sendResults(task.Sequence.ID, results, mtransitions)

			}
		}(index)
	}
}
func (exec *Executor) parseResults(alltransitions []*statecell.StateCell) map[uint64][]*statecell.StateCell {
	mTransitions := make(map[uint64][]*statecell.StateCell, len(alltransitions))
	for i := range alltransitions {
		id := alltransitions[i].GetTx()
		mTransitions[id] = append(mTransitions[id], alltransitions[i])
	}

	return mTransitions
}
func addGroupIds(groupid uint64, accessRecords statecell.StateCells) statecell.StateCells {
	for i := range accessRecords {
		accessRecords[i].Property.JobSequenceID = groupid
	}
	return accessRecords
}

func (exec *Executor) sendResults(groupId uint64, results []*workload.Result, mTransitions map[uint64][]*statecell.StateCell) {
	counter := len(results)

	exec.ctx.LogDebug("sendResult", logger.F("results counter", counter))
	sendingEuResults := make([]*eushared.EuResult, counter)
	sendingNonceEuResults := make([]*eushared.EuResult, counter)
	sendingAccessRecords := make([]*eushared.TxAccessRecords, counter)
	sendingReceipts := make([]*evmTypes.Receipt, counter)
	contractAddress := []evmCommon.Address{}
	nilAddress := evmCommon.Address{}
	sendingCallResults := make([][]byte, counter)
	txsResults := make([]*mtypes.ExecuteResponse, counter)

	threadNum := runtime.NumCPU()
	if len(results) < 100 {
		threadNum = 1
	}
	faileds := make([]int, len(results))
	contractAddresses := make([]evmCommon.Address, len(results))
	slice.ParallelForeach(results, threadNum, func(i int, result **workload.Result) {
		rawtransitions := mTransitions[(*result).TxInfo.ID]
		accesses := statecell.StateCells(slice.Clone(rawtransitions)).To(statecell.InterProcAccess{})
		transitions := statecell.StateCells(rawtransitions).To(statecell.InterProcTransition{})

		if (*result).Receipt.Status == 0 {
			faileds[i] = 1
			exec.ctx.LogErr("Tx failed", logger.F("txhash", fmt.Sprintf("%x", (*result).TxInfo.Hash[:])), logger.F("err", fmt.Sprintf("%v", (*result).Err)))
		}
		euresult := eushared.EuResult{}
		euresult.Hash = (*result).TxInfo.Hash
		euresult.GasUsed = (*result).Receipt.GasUsed
		euresult.Status = (*result).Receipt.Status
		euresult.ID = (*result).TxInfo.ID
		euresult.Trans = transitions
		sendingEuResults[i] = &euresult

		nonceEuresult := eushared.EuResult{}
		nonceEuresult.Hash = (*result).TxInfo.Hash
		nonceEuresult.Status = (*result).Receipt.Status
		nonceEuresult.ID = (*result).TxInfo.ID

		nonceTransactions := slice.CloneIf(transitions, func(v *statecell.StateCell) bool {
			path := *v.GetPath()
			return path[len(path)-5:] == "nonce"
		}, func(v *statecell.StateCell) *statecell.StateCell {
			return v.Clone().(*statecell.StateCell)
		})

		nonceEuresult.Trans = nonceTransactions

		sendingNonceEuResults[i] = &nonceEuresult

		accessRecord := eushared.TxAccessRecords{}
		accessRecord.Hash = (*result).TxInfo.Hash
		accessRecord.ID = (*result).TxInfo.ID

		accessRecord.Accesses = addGroupIds(groupId, accesses)
		sendingAccessRecords[i] = &accessRecord

		sendingReceipts[i] = (*result).Receipt

		contractAddresses[i] = nilAddress
		if (*result).Receipt.ContractAddress != nilAddress {
			contractAddresses[i] = (*result).Receipt.ContractAddress
		}

		sendingCallResults[i] = (*result).EvmResult.ReturnData

		txsResults[i] = &mtypes.ExecuteResponse{
			Hash:    euresult.Hash,
			Status:  euresult.Status,
			GasUsed: euresult.GasUsed,
		}
	})

	contractAddress = slice.CopyIf(contractAddresses, func(_ int, hash evmCommon.Address) bool {
		return hash != nilAddress
	})
	failedss := slice.CopyIf(faileds, func(_ int, flag int) bool {
		return flag == 1
	})
	exec.ctx.LogDebug("execute Results", logger.F("failed", len(failedss)))

	//-----------------------------start sending ------------------------------

	euresults := eushared.Euresults(sendingEuResults)
	exec.ctx.Send(scommon.MsgEuResults, &euresults, exec.height)
	exec.ctx.LogDebug("sendResult MsgEuResults", logger.F("euresults", len(euresults)))

	nonceeEuresults := eushared.Euresults(sendingNonceEuResults)
	exec.ctx.Send(scommon.MsgNonceEuResults, &nonceeEuresults, exec.height)
	exec.ctx.LogDebug("sendResult nonceeEuresults", logger.F("nonceeEuresults", len(nonceeEuresults)))

	responses := mtypes.JobSequenceResponse{
		Responses:       txsResults,
		ContractAddress: contractAddress,
		CallResults:     [][]byte{},
	}

	exec.resultCh <- &responses

	tarss := eushared.TxAccessRecordSet(sendingAccessRecords)
	exec.ctx.Send(scommon.MsgTxAccessRecords, &tarss, exec.height)
	exec.ctx.LogDebug("sendResult MsgTxAccessRecords", logger.F("MsgTxAccessRecords", len(tarss)))

	if counter > 0 {
		exec.ctx.Send(scommon.MsgReceipts, sendingReceipts, exec.height)
	}
}
