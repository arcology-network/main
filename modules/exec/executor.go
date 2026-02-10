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

	eupk "github.com/arcology-network/eu/common"
	cache "github.com/arcology-network/storage-committer/storage/cache"
	univaluepk "github.com/arcology-network/storage-committer/type/univalue"

	"github.com/arcology-network/common-lib/exp/mempool"
	"github.com/arcology-network/common-lib/exp/slice"

	apihandler "github.com/arcology-network/eu/apihandler"
	eucommon "github.com/arcology-network/eu/common"
	eushared "github.com/arcology-network/eu/shared"
	mtypes "github.com/arcology-network/main/types"
	evmTypes "github.com/ethereum/go-ethereum/core/types"

	statestore "github.com/arcology-network/storage-committer"

	scommon "github.com/arcology-network/streamer/common"
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
	resultCh chan *ExecutorResponse
	numTasks int
	ctx      *actor.ExecutionContext

	chainId *big.Int

	store     *statestore.StateStore
	stateInit bool

	euCount int
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
		scommon.MsgTxsToExecute,          // Txs to run.
		scommon.MsgGenerationReapingList, //update generationIdx
		scommon.MsgBlockEnd,
		scommon.MsgInitialization,
	}, false
}

func (exec *Executor) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgReceipts:          100, // Exec results.
		scommon.MsgEuResults:         100, // Exec results.
		scommon.MsgNonceEuResults:    100,
		scommon.MsgTxAccessRecords:   100, // Access records for arbitrator.
		scommon.MsgTxsExecuteResults: 1,   // To wake up rpc service.
	}
}

func (exec *Executor) Config(params map[string]interface{}) {
	exec.chainId = params["chain_id"].(*big.Int)
	exec.euCount = params["eus"].(int)
	exec.taskCh = make(chan *exetyp.ExecMessagers, exec.euCount)
	exec.resultCh = make(chan *ExecutorResponse, exec.euCount)
	exec.startExec()
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
			scommon.MsgTxsToExecute,
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
	reg.Register(scommon.MsgTxsToExecute, exec.newTxsPack)
	reg.Register(scommon.MsgGenerationReapingList, exec.receivedGenerationReapingList)
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
	exec.store = msg.Data.(*statestore.StateStore)
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
func (exec *Executor) newTxsPack(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	request := msg.Data.(*mtypes.ExecutorRequest)
	exec.numTasks = len(request.Sequences)
	exec.ctx = ctx.ExecCtx.Fork()
	if exec.generationIdx != request.GenerationIdx {
		panic("generationIdx not match!")
	}
	for i := range request.Sequences {
		exec.sendNewTask(request.Timestamp, request.Sequences[i])
	}
	exec.collectResults(ctx)
	return nil
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
	sequence *mtypes.ExecutingSequence,
	// execCtx *actor.ExecutionContext,
) {
	config := exetyp.MainConfig(exec.chainId)
	config.Coinbase = exec.execParams.Coinbase
	config.BlockNumber = new(big.Int).SetUint64(exec.height)
	config.Time = timestamp
	config.ParentHash = evmCommon.BytesToHash(exec.execParams.ParentInfo.ParentHash.Bytes())
	task := &exetyp.ExecMessagers{
		Sequence: sequence,
		Config:   config,
		// ExecCtx:  execCtx,
	}
	exec.taskCh <- task
}

func (exec *Executor) collectResults(ctx *actor.ActionContext) {
	responses := make([]*ExecutorResponse, exec.numTasks)
	for i := 0; i < exec.numTasks; i++ {
		responses[i] = <-exec.resultCh
	}
	// exec.ctx.ExecCtx.AddRpcReqID(exec.requestId)
	exec.ctx.Send(scommon.MsgTxsExecuteResults, responses, exec.height)
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
				exec.ctx.LogDebug("start execute", logger.F("Sequence.Parallel", task.Sequence.Parallel), logger.F("txs counter", len(task.Sequence.Msgs)))
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

						jobsequence.Run(task.Config, api, GetThreadID(task.Sequence.Msgs[j].TxHash))
						results = append(results, jobsequence.Jobs[0].Results)
						mtransitions[uint64(task.Sequence.Msgs[j].ID)] = jobsequence.Jobs[0].Results.Transitions()
					}
					exec.sendResults(task.Sequence.GroupIds, results, mtransitions)
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
					transitions := jobsequence.GetClearedTransition()
					mtransitions := exec.parseResults(transitions)

					results := make([]*eucommon.Result, len(task.Sequence.Msgs))
					for i := range task.Sequence.Msgs {
						results[i] = jobsequence.Jobs[i].Results
					}

					exec.sendResults(task.Sequence.GroupIds, results, mtransitions)
				}
			}
		}(index)
	}
}
func (exec *Executor) parseResults(alltransitions []*univaluepk.Univalue) map[uint64][]*univaluepk.Univalue {
	mTransitions := make(map[uint64][]*univaluepk.Univalue, len(alltransitions))
	for i := range alltransitions {
		id := alltransitions[i].GetTx()
		mTransitions[id] = append(mTransitions[id], alltransitions[i])
	}

	return mTransitions
}
func addGroupIds(groupid uint64, accessRecords univaluepk.Univalues) univaluepk.Univalues {
	for i := range accessRecords {
		accessRecords[i].Property.SetSequence(groupid)
	}
	return accessRecords
}

func (exec *Executor) sendResults(groupIds []uint64, results []*eucommon.Result, mTransitions map[uint64][]*univaluepk.Univalue) {
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
	slice.ParallelForeach(results, threadNum, func(i int, result **eucommon.Result) {
		rawtransitions := mTransitions[uint64((*result).StdMsg.ID)]
		accesses := univaluepk.Univalues(slice.Clone(rawtransitions)).To(univaluepk.IPAccess{})
		transitions := univaluepk.Univalues(rawtransitions).To(univaluepk.IPTransition{})

		// fmt.Printf("-----------------------------------main/modules/exec/executor.go--------size:%v-----\n", len(transitions))
		// univaluepk.Univalues(transitions).Print()
		// transitions.Print(func(v *univalue.Univalue) bool {
		// 	//return v.Writes() > 0 || v.DeltaWrites() > 0
		// 	return true
		// })

		// fmt.Printf("====================================main/modules/exec/executor.go================\n")
		// accesses.Print()

		if (*result).Receipt.Status == 0 {
			faileds[i] = 1
			exec.ctx.LogErr("Tx failed", logger.F("txhash", fmt.Sprintf("%x", (*result).TxHash[:])), logger.F("err", (*result).EvmResult.Err))
		}
		euresult := eushared.EuResult{}
		euresult.H = string((*result).TxHash[:])
		euresult.GasUsed = (*result).Receipt.GasUsed
		euresult.Status = (*result).Receipt.Status
		euresult.ID = uint64((*result).StdMsg.ID)
		euresult.Trans = transitions
		sendingEuResults[i] = &euresult

		nonceEuresult := eushared.EuResult{}
		nonceEuresult.H = string((*result).TxHash[:])
		nonceEuresult.Status = (*result).Receipt.Status
		nonceEuresult.ID = uint64((*result).StdMsg.ID)

		nonceTransactions := slice.CloneIf(transitions, func(v *univaluepk.Univalue) bool {
			path := *v.GetPath()
			return path[len(path)-5:] == "nonce"
		}, func(v *univaluepk.Univalue) *univaluepk.Univalue {
			return v.Clone().(*univaluepk.Univalue)
		})

		nonceEuresult.Trans = nonceTransactions //univaluepk.Univalues(nonceTransactions).Clone()

		sendingNonceEuResults[i] = &nonceEuresult

		accessRecord := eushared.TxAccessRecords{}
		accessRecord.Hash = euresult.H
		accessRecord.ID = uint64((*result).StdMsg.ID)

		accessRecord.Accesses = addGroupIds(groupIds[i], accesses)
		sendingAccessRecords[i] = &accessRecord

		sendingReceipts[i] = (*result).Receipt

		contractAddresses[i] = nilAddress
		if (*result).Receipt.ContractAddress != nilAddress {
			contractAddresses[i] = (*result).Receipt.ContractAddress
		}

		sendingCallResults[i] = (*result).EvmResult.ReturnData

		txsResults[i] = &mtypes.ExecuteResponse{
			Hash:    evmCommon.BytesToHash([]byte(euresult.H)),
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

	responses := ExecutorResponse{
		Responses:       txsResults,
		ContractAddress: contractAddress,
	}

	responses.CallResults = [][]byte{}

	exec.resultCh <- &responses

	tarss := eushared.TxAccessRecordSet(sendingAccessRecords)
	exec.ctx.Send(scommon.MsgTxAccessRecords, &tarss, exec.height)
	exec.ctx.LogDebug("sendResult MsgTxAccessRecords", logger.F("MsgTxAccessRecords", len(tarss)))

	if counter > 0 {
		exec.ctx.Send(scommon.MsgReceipts, sendingReceipts, exec.height)
	}
}
