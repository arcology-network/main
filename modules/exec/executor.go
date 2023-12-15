package exec

import (
	"math"
	"math/big"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	"github.com/arcology-network/component-lib/storage"
	ccurl "github.com/arcology-network/concurrenturl"
	"github.com/arcology-network/concurrenturl/commutative"
	"github.com/arcology-network/concurrenturl/indexer"
	"github.com/arcology-network/concurrenturl/interfaces"
	ccdb "github.com/arcology-network/concurrenturl/storage"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"

	concurrenturlcommon "github.com/arcology-network/concurrenturl/common"
	univaluepk "github.com/arcology-network/concurrenturl/univalue"
	ccapi "github.com/arcology-network/vm-adaptor/api"
	eucommon "github.com/arcology-network/vm-adaptor/common"
	"github.com/arcology-network/vm-adaptor/execution"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type ExecutorResponse struct {
	Responses       []*types.ExecuteResponse
	ContractAddress []evmCommon.Address
	CallResults     [][]byte
}

const (
	execStateWaitDBCommit = iota
	execStateWaitBlockStart
	execStateReady
	execStateProcessing
)

type pendingTask struct {
	precedings         []*evmCommon.Hash
	baseOn             *interfaces.Datastore
	totalPrecedingSize int

	sequence  *types.ExecutingSequence
	timestamp *big.Int
	debug     bool
}

func (ps *pendingTask) match(results []*types.EuResult) bool {
	if len(ps.precedings) != len(results) {
		return false
	}

	hashDict := make(map[string]struct{})
	for _, r := range results {
		hashDict[r.H] = struct{}{}
	}

	for _, p := range ps.precedings {
		if _, ok := hashDict[string(p.Bytes())]; !ok {
			return false
		}
	}
	return true
}

type Executor struct {
	actor.WorkerThread

	state  int
	height uint64

	snapshotDict SnapshotDict
	execParams   *exetyp.ExecutorParameter
	//eus          []ExecutionImpl
	apis         []eucommon.EthApiRouter
	taskCh       chan *exetyp.ExecMessagers
	resultCh     chan *ExecutorResponse
	genExecLog   bool
	pendingTasks map[evmCommon.Hash][]*pendingTask
	url          *ccurl.ConcurrentUrl
	numTasks     int
	requestId    uint64

	chainId *big.Int
}

func NewExecutor(concurrency int, groupId string) actor.IWorkerEx {
	exec := &Executor{
		state:        execStateWaitDBCommit,
		height:       math.MaxUint64,
		snapshotDict: exetyp.NewLookup(),
		apis:         make([]eucommon.EthApiRouter, concurrency),
		taskCh:       make(chan *exetyp.ExecMessagers, concurrency),
		resultCh:     make(chan *ExecutorResponse, concurrency),
		url:          ccurl.NewConcurrentUrl(nil),
	}
	exec.Set(concurrency, groupId)
	return exec
}

func (exec *Executor) Inputs() ([]string, bool) {
	return []string{
		actor.CombinedName(actor.MsgApcHandle, actor.MsgCached),      // Init DB on certain block.
		actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo), // Got block context.
		actor.MsgTxsToExecute,       // Txs to run.
		actor.MsgPrecedingsEuresult, // Preceding EuResults to make snapshot.
		actor.MsgSelectedExecuted,   // Disable Executor before local DB commit.
	}, false
}

func (exec *Executor) Outputs() map[string]int {
	return map[string]int{
		actor.MsgPrecedingList:     1,   // To collect preceding EuResults.
		actor.MsgExecuted:          1,   // To commit GeneralUrl.
		actor.MsgReceipts:          100, // Exec results.
		actor.MsgReceiptHashList:   100, // Exec results.
		actor.MsgEuResults:         100, // Exec results.
		actor.MsgTxAccessRecords:   100, // Access records for arbitrator.
		actor.MsgTxsExecuteResults: 1,   // To wake up rpc service.
		actor.MsgExecutingLogs:     1,   // Exec logs.
	}
}

func (exec *Executor) Config(params map[string]interface{}) {
	if v, ok := params["gatherexeclog"]; ok {
		exec.genExecLog = v.(bool)
	}
	exec.chainId = params["chain_id"].(*big.Int)
}

func (exec *Executor) OnStart() {
	for i := 0; i < int(exec.Concurrency); i++ {
		persistentDB := ccdb.NewParallelEthMemDataStore() //cachedstorage.NewDataStore()
		persistentDB.Inject(concurrenturlcommon.ETH10_ACCOUNT_PREFIX, commutative.NewPath())
		db := ccdb.NewTransientDB(persistentDB)

		url := ccurl.NewConcurrentUrl(db)

		api := ccapi.NewAPI(url)

		exec.apis[i] = api

	}
	exec.startExec()
}

func (exec *Executor) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch exec.state {
	case execStateWaitDBCommit:
		db := msg.Data.(*actor.CombinerElements).Get(actor.MsgApcHandle).Data.(*interfaces.Datastore)
		exec.snapshotDict.Reset(db)
		exec.height = msg.Height + 1
		exec.state = execStateWaitBlockStart
	case execStateWaitBlockStart:
		combined := msg.Data.(*actor.CombinerElements)
		coinbase := evmCommon.BytesToAddress(combined.Get(actor.MsgBlockStart).Data.(*actor.BlockStart).Coinbase.Bytes())
		exec.execParams = &exetyp.ExecutorParameter{
			ParentInfo: combined.Get(actor.MsgParentInfo).Data.(*types.ParentInfo),
			Coinbase:   &coinbase,
			Height:     combined.Get(actor.MsgBlockStart).Height,
		}
		exec.state = execStateReady
	case execStateReady:
		switch msg.Name {
		case actor.MsgTxsToExecute:
			request := msg.Data.(*types.ExecutorRequest)
			exec.numTasks = len(request.Sequences)
			exec.requestId = msg.Msgid
			exec.pendingTasks = make(map[evmCommon.Hash][]*pendingTask)
			for i := range request.Sequences {
				snapshot, precedings := exec.snapshotDict.Query(request.Precedings[i])
				if len(precedings) == 0 {
					// Available snapshot found.
					exec.sendNewTask(*snapshot, request.Timestamp, request.Sequences[i], request.Debug)
				} else {
					exec.pendingTasks[request.PrecedingHash[i]] = append(exec.pendingTasks[request.PrecedingHash[i]], &pendingTask{
						precedings:         precedings,
						baseOn:             snapshot,
						totalPrecedingSize: len(request.Precedings[i]),
						sequence:           request.Sequences[i],
						timestamp:          request.Timestamp,
						debug:              request.Debug,
					})
				}
			}

			if len(exec.pendingTasks) > 0 {
				for _, pt := range exec.pendingTasks {
					exec.MsgBroker.Send(actor.MsgPrecedingList, &pt[0].precedings, exec.height)
				}
				exec.state = execStateProcessing
			} else {
				exec.collectResults()
			}
		case actor.MsgSelectedExecuted:
			exec.MsgBroker.Send(actor.MsgExecuted, msg.Data, msg.Height)
			exec.state = execStateWaitDBCommit
		}
	case execStateProcessing:
		data := msg.Data.([]interface{})
		results := make([]*types.EuResult, len(data))
		for i, d := range data {
			results[i] = d.(*types.EuResult)
		}
		for hash, pt := range exec.pendingTasks {
			if pt[0].match(results) {
				// The following code were copied from exec v1.
				db := ccdb.NewTransientDB(*pt[0].baseOn)
				exec.url.Init(db)
				txIds, transitions := storage.GetTransitions(results)
				exec.url.Import(transitions)
				exec.url.Sort()
				exec.url.Commit(txIds)
				exec.snapshotDict.AddItem(hash, pt[0].totalPrecedingSize, &db)

				for _, task := range pt {
					exec.sendNewTask(db, task.timestamp, task.sequence, task.debug)
				}
				delete(exec.pendingTasks, hash)
				break
			}
		}
		if len(exec.pendingTasks) == 0 {
			exec.collectResults()
			exec.state = execStateReady
		}
	}
	return nil
}

func (exec *Executor) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		execStateWaitDBCommit:   {actor.CombinedName(actor.MsgApcHandle, actor.MsgCached)},
		execStateWaitBlockStart: {actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo)},
		execStateReady: {
			actor.MsgTxsToExecute,
			actor.MsgSelectedExecuted,
		},
		execStateProcessing: {
			actor.MsgPrecedingsEuresult,
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
	snapshot interfaces.Datastore,
	timestamp *big.Int,
	sequence *types.ExecutingSequence,
	debug bool,
) {
	//db := curstorage.NewTransientDB(snapshot)
	config := exetyp.MainConfig(exec.chainId)
	config.Coinbase = exec.execParams.Coinbase
	config.BlockNumber = new(big.Int).SetUint64(exec.height)
	config.Time = timestamp
	config.ParentHash = evmCommon.BytesToHash(exec.execParams.ParentInfo.ParentHash.Bytes())
	task := &exetyp.ExecMessagers{
		Sequence: sequence,
		Snapshot: &snapshot,
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

func toJobSequence(msgs []*types.StandardMessage, ids []uint32) []*execution.StandardMessage {
	nmsgs := make([]*execution.StandardMessage, len(msgs))
	for i, msg := range msgs {
		msg.Native.SkipAccountChecks = true
		nmsgs[i] = &execution.StandardMessage{
			TxHash: msg.TxHash,
			Native: msg.Native,
			Source: msg.Source,
			ID:     uint64(ids[i]),
		}
	}
	return nmsgs
}

func (exec *Executor) startExec() {
	for i := 0; i < int(exec.Concurrency); i++ {
		index := i
		go func(index int) {
			for {
				task := <-exec.taskCh
				if task.Sequence.Parallel {
					results := make([]*execution.Result, 0, len(task.Sequence.Msgs))
					for j := range task.Sequence.Msgs {
						job := execution.JobSequence{
							ID:        uint32(j),
							StdMsgs:   toJobSequence([]*types.StandardMessage{task.Sequence.Msgs[j]}, []uint32{task.Sequence.Txids[j]}),
							ApiRouter: exec.apis[index],
						}

						url := ccurl.NewConcurrentUrl(*task.Snapshot)
						api := ccapi.NewAPI(url)
						job.Run(task.Config, api)

						results = append(results, job.Results...)
					}
					exec.sendResults(results, task.Sequence.Txids, task.Debug)
				} else {
					job := execution.JobSequence{
						ID:        uint32(0),
						StdMsgs:   toJobSequence(task.Sequence.Msgs, task.Sequence.Txids),
						ApiRouter: exec.apis[index],
					}

					url := ccurl.NewConcurrentUrl(*task.Snapshot)
					api := ccapi.NewAPI(url)
					job.Run(task.Config, api)

					exec.sendResults(job.Results, task.Sequence.Txids, task.Debug)
				}

			}
		}(index)
	}
}
func (exec *Executor) sendResults(results []*execution.Result, txids []uint32, debug bool) {
	counter := len(results)
	exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult", zap.Bool("debug", debug), zap.Int("results counter", counter))
	sendingEuResults := make([]*types.EuResult, counter)
	sendingAccessRecords := make([]*types.TxAccessRecords, counter)
	sendingReceipts := make([]*evmTypes.Receipt, counter)
	contractAddress := []evmCommon.Address{}
	nilAddress := evmCommon.Address{}
	sendingCallResults := make([][]byte, counter)
	txsResults := make([]*types.ExecuteResponse, counter)
	failed := 0
	for i, result := range results {
		accesses := indexer.Univalues(common.Clone(result.Transitions())).To(indexer.ITCAccess{})
		transitions := indexer.Univalues(common.Clone(result.Transitions())).To(indexer.ITCTransition{})

		if result.Receipt.Status == 0 {
			failed++
		}
		euresult := types.EuResult{}
		euresult.H = string(result.TxHash[:])
		euresult.GasUsed = result.Receipt.GasUsed
		euresult.Status = result.Receipt.Status
		euresult.ID = txids[i]
		euresult.Transitions = univaluepk.UnivaluesEncode(transitions)
		sendingEuResults[i] = &euresult

		accessRecord := types.TxAccessRecords{}
		accessRecord.Hash = euresult.H
		accessRecord.ID = txids[i]
		accessRecord.Accesses = univaluepk.UnivaluesEncode(accesses)
		sendingAccessRecords[i] = &accessRecord

		sendingReceipts[i] = result.Receipt

		if result.Receipt.ContractAddress != nilAddress {
			contractAddress = append(contractAddress, result.Receipt.ContractAddress)
		}

		sendingCallResults[i] = result.EvmResult.ReturnData

		txsResults[i] = &types.ExecuteResponse{
			Hash:    evmCommon.BytesToHash([]byte(euresult.H)),
			Status:  euresult.Status,
			GasUsed: euresult.GasUsed,
		}
	}
	exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>execute Results", zap.Int("failed", failed))
	//-----------------------------start sending ------------------------------
	if !debug {
		euresults := types.Euresults(sendingEuResults)
		exec.MsgBroker.Send(actor.MsgEuResults, &euresults)
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
		tarss := types.TxAccessRecordSet(sendingAccessRecords)
		exec.MsgBroker.Send(actor.MsgTxAccessRecords, &tarss)
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult MsgTxAccessRecords", zap.Int("MsgTxAccessRecords", len(tarss)))
	}
	if debug {
		return
	}

	if counter > 0 {
		exec.MsgBroker.Send(actor.MsgReceipts, &sendingReceipts)
		receiptHashList := exec.toReceiptsHash(&sendingReceipts)
		exec.MsgBroker.Send(actor.MsgReceiptHashList, receiptHashList)
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult MsgReceiptHashList", zap.Int("receiptHashList", len(receiptHashList.TxHashList)))
	}
}

func (e *Executor) toReceiptsHash(receipts *[]*evmTypes.Receipt) *types.ReceiptHashList {
	// The following code were copied from exec v1.
	rcptLength := 0
	if receipts != nil {
		rcptLength = len(*receipts)
	}

	if rcptLength == 0 {
		return &types.ReceiptHashList{}
	}

	receiptHashList := make([]evmCommon.Hash, rcptLength)
	txHashList := make([]evmCommon.Hash, rcptLength)
	gasUsedList := make([]uint64, rcptLength)
	worker := func(start, end, idx int, args ...interface{}) {
		receipts := args[0].([]interface{})[0].([]*evmTypes.Receipt)
		receiptHashList := args[0].([]interface{})[1].([]evmCommon.Hash)
		txHashList := args[0].([]interface{})[2].([]evmCommon.Hash)
		gasUsedList := args[0].([]interface{})[3].([]uint64)

		for i := range receipts[start:end] {
			idx := i + start
			receipt := receipts[idx]
			txHashList[idx] = receipt.TxHash
			receiptHashList[idx] = types.RlpHash(receipt)
			gasUsedList[idx] = receipt.GasUsed
		}
	}
	common.ParallelWorker(len(*receipts), e.Concurrency, worker, *receipts, receiptHashList, txHashList, gasUsedList)
	return &types.ReceiptHashList{
		TxHashList:      txHashList,
		ReceiptHashList: receiptHashList,
		GasUsedList:     gasUsedList,
	}
}
