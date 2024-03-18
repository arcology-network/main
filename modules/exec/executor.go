package exec

import (
	"math"
	"math/big"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/main/components/storage"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	ccurl "github.com/arcology-network/storage-committer"
	"github.com/arcology-network/storage-committer/commutative"
	"github.com/arcology-network/storage-committer/interfaces"
	ccdb "github.com/arcology-network/storage-committer/storage"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"

	eupk "github.com/arcology-network/eu"
	"github.com/arcology-network/eu/cache"
	concurrenturlcommon "github.com/arcology-network/storage-committer/common"
	"github.com/arcology-network/storage-committer/importer"
	univaluepk "github.com/arcology-network/storage-committer/univalue"

	"github.com/arcology-network/common-lib/exp/mempool"
	"github.com/arcology-network/common-lib/exp/slice"
	eucommon "github.com/arcology-network/eu/common"
	"github.com/arcology-network/eu/execution"
	eushared "github.com/arcology-network/eu/shared"
	apihandler "github.com/arcology-network/evm-adaptor/apihandler"
	intf "github.com/arcology-network/evm-adaptor/interface"
	mtypes "github.com/arcology-network/main/types"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type ExecutorResponse struct {
	Responses       []*mtypes.ExecuteResponse
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

	sequence  *mtypes.ExecutingSequence
	timestamp *big.Int
	debug     bool
}

func (ps *pendingTask) match(results []*eushared.EuResult) bool {
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

	snapshotDict   SnapshotDict
	execParams     *exetyp.ExecutorParameter
	apis           []intf.EthApiRouter
	taskCh         chan *exetyp.ExecMessagers
	resultCh       chan *ExecutorResponse
	genExecLog     bool
	pendingTasks   map[evmCommon.Hash][]*pendingTask
	stateCommitter *ccurl.StateCommitter
	numTasks       int
	requestId      uint64

	chainId *big.Int
}

func NewExecutor(concurrency int, groupId string) actor.IWorkerEx {
	exec := &Executor{
		state:          execStateWaitDBCommit,
		height:         math.MaxUint64,
		snapshotDict:   exetyp.NewLookup(),
		apis:           make([]intf.EthApiRouter, concurrency),
		taskCh:         make(chan *exetyp.ExecMessagers, concurrency),
		resultCh:       make(chan *ExecutorResponse, concurrency),
		stateCommitter: ccurl.NewStorageCommitter(nil),
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
		actor.MsgPrecedingList: 1,   // To collect preceding EuResults.
		actor.MsgExecuted:      1,   // To commit GeneralUrl.
		actor.MsgReceipts:      100, // Exec results.
		// actor.MsgReceiptHashList:   100, // Exec results.
		actor.MsgEuResults:         100, // Exec results.
		actor.MsgTxAccessRecords:   100, // Access records for arbitrator.
		actor.MsgTxsExecuteResults: 1,   // To wake up rpc service.
		// actor.MsgExecutingLogs:     1,   // Exec logs.
	}
}

func (exec *Executor) Config(params map[string]interface{}) {
	exec.chainId = params["chain_id"].(*big.Int)
}

func (exec *Executor) OnStart() {
	for i := 0; i < int(exec.Concurrency); i++ {
		persistentDB := ccdb.NewParallelEthMemDataStore() //cachedstorage.NewDataStore()
		persistentDB.Inject(concurrenturlcommon.ETH10_ACCOUNT_PREFIX, commutative.NewPath())
		db := ccdb.NewTransientDB(persistentDB)
		// localCache := cache.NewWriteCache(db)
		// api := apihandler.NewAPIHandler(localCache)

		api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
			return cache.NewWriteCache(db, 32, 1)
		}, (&cache.WriteCache{}).Reset))

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
			ParentInfo: combined.Get(actor.MsgParentInfo).Data.(*mtypes.ParentInfo),
			Coinbase:   &coinbase,
			Height:     combined.Get(actor.MsgBlockStart).Height,
		}
		exec.state = execStateReady
	case execStateReady:
		switch msg.Name {
		case actor.MsgTxsToExecute:
			request := msg.Data.(*mtypes.ExecutorRequest)
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
		results := make([]*eushared.EuResult, len(data))
		for i, d := range data {
			results[i] = d.(*eushared.EuResult)
		}
		for hash, pt := range exec.pendingTasks {
			if pt[0].match(results) {
				// The following code were copied from exec v1.
				db := ccdb.NewTransientDB(*pt[0].baseOn)
				exec.stateCommitter.Init(db)
				ids, transitions := storage.GetTransitions(results)
				exec.stateCommitter.Import(transitions)
				exec.stateCommitter.Sort()
				// exec.stateCommitter.CopyToDbBuffer()
				exec.stateCommitter.Precommit(ids)
				exec.stateCommitter.Commit()
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
	sequence *mtypes.ExecutingSequence,
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

//	func toJobSequence(msgs []*eucommon.StandardMessage, ids []uint32) []*eucommon.StandardMessage {
//		nmsgs := make([]*eucommon.StandardMessage, len(msgs))
//		for i, msg := range msgs {
//			msg.NativeMessage.SkipAccountChecks = true
//			nmsgs[i] = &eucommon.StandardMessage{
//				TxHash: msg.TxHash,
//				Native: msg.NativeMessage,
//				Source: msg.Source,
//				ID:     uint64(ids[i]),
//			}
//		}
//		return nmsgs
//	}
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
							ID:        uint32(j),
							StdMsgs:   []*eucommon.StandardMessage{task.Sequence.Msgs[j]},
							ApiRouter: exec.apis[index],
						}

						api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
							return cache.NewWriteCache(*task.Snapshot, 32, 1)
						}, (&cache.WriteCache{}).Reset))

						job.Run(task.Config, api, GetThreadD(job.StdMsgs[0].TxHash))
						results = append(results, job.Results...)
					}
					exec.sendResults(results, task.Sequence.Txids, task.Debug)
				} else {
					job := eupk.JobSequence{
						ID:        uint32(0),
						StdMsgs:   task.Sequence.Msgs,
						ApiRouter: exec.apis[index],
					}

					api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
						return cache.NewWriteCache(*task.Snapshot, 32, 1)
					}, (&cache.WriteCache{}).Reset))

					job.Run(task.Config, api, GetThreadD(job.StdMsgs[0].TxHash))

					exec.sendResults(job.Results, task.Sequence.Txids, task.Debug)
				}
				// exec.AddLog(log.LogLevel_Debug, "Task Style", zap.Duration("exectime", time.Since(timeStart)), zap.Bool("Parallel", task.Sequence.Parallel), zap.String("SequenceId", fmt.Sprintf("%x", task.Sequence.SequenceId.Bytes())), zap.Int("Size", len(task.Sequence.Msgs)))
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
	for i, result := range results {
		accesses := univaluepk.Univalues(slice.Clone(result.Transitions())).To(importer.ITAccess{})
		transitions := univaluepk.Univalues(slice.Clone(result.Transitions())).To(importer.ITTransition{})

		// fmt.Printf("-----------------------------------main/modules/exec/executor.go-------------\n")
		// transitions.Print()

		// fmt.Printf("====================================main/modules/exec/executor.go================\n")
		// accesses.Print()

		if result.Receipt.Status == 0 {
			failed++
		}
		euresult := eushared.EuResult{}
		euresult.H = string(result.TxHash[:])
		euresult.GasUsed = result.Receipt.GasUsed
		euresult.Status = result.Receipt.Status
		euresult.ID = txids[i]
		euresult.Transitions = common.IfThen(len(transitions.Keys()) == 0, []byte{}, transitions.Encode())
		sendingEuResults[i] = &euresult

		accessRecord := eushared.TxAccessRecords{}
		accessRecord.Hash = euresult.H
		accessRecord.ID = txids[i]

		accessRecord.Accesses = common.IfThen(len(accesses.Keys()) == 0, []byte{}, accesses.Encode())
		sendingAccessRecords[i] = &accessRecord

		sendingReceipts[i] = result.Receipt
		if result.Receipt.ContractAddress != nilAddress {
			contractAddress = append(contractAddress, result.Receipt.ContractAddress)
		}

		sendingCallResults[i] = result.EvmResult.ReturnData

		txsResults[i] = &mtypes.ExecuteResponse{
			Hash:    evmCommon.BytesToHash([]byte(euresult.H)),
			Status:  euresult.Status,
			GasUsed: euresult.GasUsed,
		}
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
