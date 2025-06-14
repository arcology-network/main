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
	"runtime"

	"github.com/arcology-network/common-lib/codec"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"

	eupk "github.com/arcology-network/eu"
	cache "github.com/arcology-network/storage-committer/storage/tempcache"
	"github.com/arcology-network/storage-committer/type/univalue"
	univaluepk "github.com/arcology-network/storage-committer/type/univalue"

	"github.com/arcology-network/common-lib/exp/mempool"
	"github.com/arcology-network/common-lib/exp/slice"

	"github.com/arcology-network/common-lib/types"
	apihandler "github.com/arcology-network/eu/apihandler"
	eucommon "github.com/arcology-network/eu/common"
	eushared "github.com/arcology-network/eu/shared"
	mtypes "github.com/arcology-network/main/types"
	evmTypes "github.com/ethereum/go-ethereum/core/types"

	statestore "github.com/arcology-network/storage-committer"
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
	actor.WorkerThread

	state         int
	height        uint64
	generationIdx uint32

	// snapshotDict SnapshotDict
	execParams *exetyp.ExecutorParameter

	taskCh    chan *exetyp.ExecMessagers
	resultCh  chan *ExecutorResponse
	numTasks  int
	requestId uint64

	chainId *big.Int

	store     *statestore.StateStore
	stateInit bool
}

func NewExecutor(concurrency int, groupId string) actor.IWorkerEx {
	exec := &Executor{
		state:     execStateInit,
		height:    math.MaxUint64,
		taskCh:    make(chan *exetyp.ExecMessagers, concurrency),
		resultCh:  make(chan *ExecutorResponse, concurrency),
		stateInit: false,
	}
	exec.Set(concurrency, groupId)
	return exec
}

func (exec *Executor) Inputs() ([]string, bool) {
	return []string{
		actor.MsgApcHandle, // Init DB on every generation.
		actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo, actor.MsgObjectCached), // Got block context.
		actor.MsgTxsToExecute,          // Txs to run.
		actor.MsgGenerationReapingList, //update generationIdx
		actor.MsgBlockEnd,
		actor.MsgInitialization,
	}, false
}

func (exec *Executor) Outputs() map[string]int {
	return map[string]int{

		actor.MsgReceipts:          100, // Exec results.
		actor.MsgEuResults:         100, // Exec results.
		actor.MsgNonceEuResults:    100,
		actor.MsgTxAccessRecords:   100, // Access records for arbitrator.
		actor.MsgTxsExecuteResults: 1,   // To wake up rpc service.
	}
}

func (exec *Executor) Config(params map[string]interface{}) {
	exec.chainId = params["chain_id"].(*big.Int)
}

func (exec *Executor) OnStart() {

	exec.startExec()
}

func (exec *Executor) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch exec.state {
	case execStateWaitBlockStart:
		combined := msg.Data.(*actor.CombinerElements)
		coinbase := evmCommon.BytesToAddress(combined.Get(actor.MsgBlockStart).Data.(*actor.BlockStart).Coinbase.Bytes())
		exec.execParams = &exetyp.ExecutorParameter{
			ParentInfo: combined.Get(actor.MsgParentInfo).Data.(*mtypes.ParentInfo),
			Coinbase:   &coinbase,
			Height:     exec.height, //combined.Get(actor.MsgBlockStart).Height,
		}

		exec.state = execStateWaitGenerationReady

		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>state change into execStateWaitGenerationReady")
	case execStateWaitGenerationReady:
		exec.store = msg.Data.(*statestore.StateStore)
		exec.state = execStateReady
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>state change into execStateReady")
	case execStateInit:
		initialization := msg.Data.(*mtypes.Initialization)
		exec.store = initialization.Store
		exec.height = msg.Height + 1

		addr := evmCommon.BytesToAddress(initialization.BlockStart.Coinbase.Bytes())
		exec.execParams = &exetyp.ExecutorParameter{
			ParentInfo: initialization.ParentInformation,
			Coinbase:   &addr,
			Height:     exec.height,
		}

		exec.state = execStateReady
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>state change into execStateReady")
	case execStateReady:
		switch msg.Name {
		case actor.MsgTxsToExecute:
			request := msg.Data.(*mtypes.ExecutorRequest)
			exec.numTasks = len(request.Sequences)
			exec.requestId = msg.Msgid
			if exec.generationIdx != request.GenerationIdx {
				panic("generationIdx not match!")
			}
			for i := range request.Sequences {
				exec.sendNewTask(request.Timestamp, request.Sequences[i], request.Debug)
			}
			exec.collectResults()
		case actor.MsgGenerationReapingList:
			reapinglist := msg.Data.(*types.InclusiveList)
			exec.generationIdx = reapinglist.GenerationIdx
			if reapinglist.GenerationIdx > 0 {
				exec.state = execStateWaitGenerationReady
				exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>state change into execStateWaitGenerationReady")
			} else {
				exec.state = execStateNextHeight
				exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>state change into execStateNextHeight", zap.Uint64("exec.height", exec.height), zap.Uint64("msg.Height", msg.Height))
			}
		}
	case execStateNextHeight:
		exec.height = msg.Height + 1
		exec.state = execStateWaitBlockStart
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>state change into execStateWaitBlockStart", zap.Uint64("exec.height", exec.height), zap.Uint64("msg.Height", msg.Height))
	}
	return nil
}

func (exec *Executor) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		execStateWaitBlockStart: {
			actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo, actor.MsgObjectCached),
		},
		execStateWaitGenerationReady: {
			actor.MsgApcHandle,
		},
		execStateReady: {
			actor.MsgTxsToExecute,
			actor.MsgGenerationReapingList,
		},
		execStateInit: {
			actor.MsgInitialization,
		},
		execStateNextHeight: {
			actor.MsgBlockEnd,
		},
	}
}

func (exec *Executor) GetCurrentState() int {
	return exec.state
}

func (exec *Executor) Height() uint64 {
	return exec.height
}

func (exec *Executor) sendNewTask(
	timestamp *big.Int,
	sequence *mtypes.ExecutingSequence,
	debug bool,
) {
	config := exetyp.MainConfig(exec.chainId)
	config.Coinbase = exec.execParams.Coinbase
	config.BlockNumber = new(big.Int).SetUint64(exec.height)
	config.Time = timestamp
	config.ParentHash = evmCommon.BytesToHash(exec.execParams.ParentInfo.ParentHash.Bytes())
	task := &exetyp.ExecMessagers{
		Sequence: sequence,
		Config:   config,
		Debug:    debug,
	}
	exec.taskCh <- task
}
func (exec *Executor) collectResults() {
	responses := make([]*ExecutorResponse, exec.numTasks)
	for i := 0; i < exec.numTasks; i++ {
		responses[i] = <-exec.resultCh
	}
	exec.MsgBroker.Send(actor.MsgTxsExecuteResults, responses, exec.height, exec.requestId)
}
func GetThreadID(hash evmCommon.Hash) uint64 {
	return uint64(codec.Uint64(0).Decode(hash.Bytes()[:8]).(codec.Uint64))
}
func (exec *Executor) startExec() {
	for i := 0; i < int(exec.Concurrency); i++ {
		index := i
		go func(index int) {
			for {
				task := <-exec.taskCh
				exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>start execute", zap.Bool("Sequence.Parallel", task.Sequence.Parallel), zap.Int("txs counter", len(task.Sequence.Msgs)))
				if task.Sequence.Parallel {
					results := make([]*eucommon.Result, 0, len(task.Sequence.Msgs))
					mtransitions := make(map[uint64][]*univalue.Univalue, len(task.Sequence.Msgs))
					for j := range task.Sequence.Msgs {
						job := eupk.JobSequence{
							ID:      uint64(j),
							StdMsgs: []*types.StandardMessage{task.Sequence.Msgs[j]},
						}
						api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
							return exec.store.WriteCache
						}, func(cache *cache.WriteCache) { cache.Clear() }))
						job.Run(task.Config, api, GetThreadID(job.StdMsgs[0].TxHash))
						results = append(results, job.Results...)
						mtransitions[uint64(task.Sequence.Msgs[j].ID)] = job.Results[0].Transitions()
					}
					exec.sendResults(task.Sequence.GroupIds, results, mtransitions, task.Debug)
				} else {
					job := eupk.JobSequence{
						ID:      uint64(0),
						StdMsgs: task.Sequence.Msgs,
					}
					api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
						return exec.store.WriteCache
					}, func(cache *cache.WriteCache) { cache.Clear() }))
					job.Run(task.Config, api, GetThreadID(job.StdMsgs[0].TxHash))
					transitions := job.GetClearedTransition()
					mtransitions := exec.parseResults(transitions)
					exec.sendResults(task.Sequence.GroupIds, job.Results, mtransitions, task.Debug)
				}
			}
		}(index)
	}
}
func (exec *Executor) parseResults(alltransitions []*univalue.Univalue) map[uint64][]*univalue.Univalue {
	mTransitions := make(map[uint64][]*univalue.Univalue, len(alltransitions))
	for i := range alltransitions {
		id := alltransitions[i].GetTx()
		mTransitions[id] = append(mTransitions[id], alltransitions[i])
	}

	return mTransitions
}
func addGroupIds(groupid uint64, accessRecords univalue.Univalues) univalue.Univalues {
	for i := range accessRecords {
		accessRecords[i].Setsequence(groupid)
	}
	return accessRecords
}

func (exec *Executor) sendResults(groupIds []uint64, results []*eucommon.Result, mTransitions map[uint64][]*univalue.Univalue, debug bool) {
	counter := len(results)
	exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult", zap.Bool("debug", debug), zap.Int("results counter", counter))
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
	exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>execute Results", zap.Int("failed", len(failedss)))

	//-----------------------------start sending ------------------------------
	if !debug {
		euresults := eushared.Euresults(sendingEuResults)
		exec.MsgBroker.Send(actor.MsgEuResults, &euresults, exec.height)
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult MsgEuResults", zap.Int("euresults", len(euresults)))

		nonceeEuresults := eushared.Euresults(sendingNonceEuResults)
		exec.MsgBroker.Send(actor.MsgNonceEuResults, &nonceeEuresults, exec.height)
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult nonceeEuresults", zap.Int("nonceeEuresults", len(nonceeEuresults)))
	}
	responses := ExecutorResponse{
		Responses:       txsResults,
		ContractAddress: contractAddress,
	}
	if debug {
		responses.CallResults = sendingCallResults
	} else {
		responses.CallResults = [][]byte{}
	}
	exec.resultCh <- &responses

	if !debug {
		tarss := eushared.TxAccessRecordSet(sendingAccessRecords)
		exec.MsgBroker.Send(actor.MsgTxAccessRecords, &tarss, exec.height)
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult MsgTxAccessRecords", zap.Int("MsgTxAccessRecords", len(tarss)))
	}
	if debug {
		return
	}

	if counter > 0 {
		exec.MsgBroker.Send(actor.MsgReceipts, &sendingReceipts, exec.height)
	}
}
