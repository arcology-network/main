package exec

import (
	"math"
	"math/big"
	"runtime"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/common"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"

	eupk "github.com/arcology-network/eu"
	"github.com/arcology-network/storage-committer/importer"
	cache "github.com/arcology-network/storage-committer/storage/writecache"
	univaluepk "github.com/arcology-network/storage-committer/univalue"

	"github.com/arcology-network/common-lib/exp/mempool"
	"github.com/arcology-network/common-lib/exp/slice"
	eucommon "github.com/arcology-network/eu/common"
	"github.com/arcology-network/eu/execution"
	eushared "github.com/arcology-network/eu/shared"
	apihandler "github.com/arcology-network/evm-adaptor/apihandler"
	mtypes "github.com/arcology-network/main/types"
	evmTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/arcology-network/storage-committer/storage/statestore"

	"github.com/arcology-network/common-lib/types"
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
	execStateProcessing
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

	store *statestore.StateStore
}

func NewExecutor(concurrency int, groupId string) actor.IWorkerEx {
	exec := &Executor{
		state:    execStateWaitGenerationReady,
		height:   math.MaxUint64,
		taskCh:   make(chan *exetyp.ExecMessagers, concurrency),
		resultCh: make(chan *ExecutorResponse, concurrency),
	}
	exec.Set(concurrency, groupId)
	return exec
}

func (exec *Executor) Inputs() ([]string, bool) {
	return []string{
		actor.MsgApcHandle, // Init DB on every generation.
		actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo), // Got block context.
		actor.MsgTxsToExecute,          // Txs to run.
		actor.MsgGenerationReapingList, //update generationIdx
	}, false
}

func (exec *Executor) Outputs() map[string]int {
	return map[string]int{

		actor.MsgReceipts:          100, // Exec results.
		actor.MsgEuResults:         100, // Exec results.
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
			Height:     combined.Get(actor.MsgBlockStart).Height,
		}
		exec.state = execStateWaitGenerationReady
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>state change into execStateWaitGenerationReady")
	case execStateWaitGenerationReady:
		exec.store = msg.Data.(*statestore.StateStore)
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
				exec.state = execStateWaitBlockStart
				exec.height = msg.Height + 1
				exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>state change into execStateWaitBlockStart")
			}

		}
	}
	return nil
}

func (exec *Executor) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		execStateWaitBlockStart: {
			actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo),
		},
		execStateWaitGenerationReady: {
			actor.MsgApcHandle,
		},
		execStateReady: {
			actor.MsgTxsToExecute,
			actor.MsgGenerationReapingList,
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
func GetThreadD(hash evmCommon.Hash) uint64 {
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
					results := make([]*execution.Result, 0, len(task.Sequence.Msgs))
					for j := range task.Sequence.Msgs {
						job := eupk.JobSequence{
							ID:      uint32(j),
							StdMsgs: []*eucommon.StandardMessage{task.Sequence.Msgs[j]},
						}
						api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
							return exec.store.WriteCache
						}, func(cache *cache.WriteCache) { cache.Clear() }))
						job.Run(task.Config, api, GetThreadD(job.StdMsgs[0].TxHash))
						results = append(results, job.Results...)
					}
					exec.sendResults(results, task.Sequence.Txids, task.Debug)
				} else {
					job := eupk.JobSequence{
						ID:      uint32(0),
						StdMsgs: task.Sequence.Msgs,
					}
					api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
						return exec.store.WriteCache
					}, func(cache *cache.WriteCache) { cache.Clear() }))
					job.Run(task.Config, api, GetThreadD(job.StdMsgs[0].TxHash))
					exec.sendResults(job.Results, task.Sequence.Txids, task.Debug)
				}
			}
		}(index)
	}
}
func (exec *Executor) sendResults(results []*execution.Result, txids []uint32, debug bool) {
	counter := len(results)
	exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult", zap.Bool("debug", debug), zap.Int("results counter", counter))
	sendingEuResults := make([]*eushared.EuResult, counter)
	sendingAccessRecords := make([]*eushared.TxAccessRecords, counter)
	sendingReceipts := make([]*evmTypes.Receipt, counter)
	contractAddress := []evmCommon.Address{}
	nilAddress := evmCommon.Address{}
	sendingCallResults := make([][]byte, counter)
	txsResults := make([]*mtypes.ExecuteResponse, counter)
	failed := 0

	threadNum := runtime.NumCPU()
	if len(results) < 100 {
		threadNum = 1
	}
	faileds := make([]int, threadNum)
	contractAddresses := make([][]evmCommon.Address, threadNum)
	slice.ParallelForeach(results, threadNum, func(i int, result **execution.Result) {
		accesses := univaluepk.Univalues(slice.Clone((*result).Transitions())).To(importer.ITAccess{})
		transitions := univaluepk.Univalues(slice.Clone((*result).Transitions())).To(importer.ITTransition{})

		// transitions := univaluepk.Univalues(slice.Clone(result.Transitions())).To(importer.ITTransition{})
		// tms[1] = time.Since(t0)
		// fmt.Printf("-----------------------------------main/modules/exec/executor.go--------size:%v-----\n", len(transitions))
		// transitions.Print()
		// transitions.Print(func(v *univalue.Univalue) bool {
		// 	//return v.Writes() > 0 || v.DeltaWrites() > 0
		// 	return true
		// })

		// fmt.Printf("====================================main/modules/exec/executor.go================\n")
		// accesses.Print()

		if (*result).Receipt.Status == 0 {
			faileds[i]++
		}
		euresult := eushared.EuResult{}
		euresult.H = string((*result).TxHash[:])
		euresult.GasUsed = (*result).Receipt.GasUsed
		euresult.Status = (*result).Receipt.Status
		euresult.ID = txids[i]
		euresult.Transitions = common.IfThen(len(transitions.Keys()) == 0, []byte{}, transitions.Encode())
		sendingEuResults[i] = &euresult

		accessRecord := eushared.TxAccessRecords{}
		accessRecord.Hash = euresult.H
		accessRecord.ID = txids[i]

		accessRecord.Accesses = common.IfThen(len(accesses.Keys()) == 0, []byte{}, accesses.Encode())
		sendingAccessRecords[i] = &accessRecord

		sendingReceipts[i] = (*result).Receipt
		if (*result).Receipt.ContractAddress != nilAddress {
			contractAddresses[i] = append(contractAddresses[i], (*result).Receipt.ContractAddress)
		}

		sendingCallResults[i] = (*result).EvmResult.ReturnData

		txsResults[i] = &mtypes.ExecuteResponse{
			Hash:    evmCommon.BytesToHash([]byte(euresult.H)),
			Status:  euresult.Status,
			GasUsed: euresult.GasUsed,
		}
	})
	contractAddress = slice.Flatten(contractAddresses)
	for i := range faileds {
		failed += faileds[i]
	}
	exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>execute Results", zap.Int("failed", failed))
	//-----------------------------start sending ------------------------------
	if !debug {
		euresults := eushared.Euresults(sendingEuResults)
		exec.MsgBroker.Send(actor.MsgEuResults, &euresults, exec.height)
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult MsgEuResults", zap.Int("euresults", len(euresults)))
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
