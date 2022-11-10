package exec

import (
	"fmt"
	"math"
	"math/big"

	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	ethtyp "github.com/arcology-network/3rd-party/eth/types"
	cachedstorage "github.com/arcology-network/common-lib/cachedstorage"
	cmncmn "github.com/arcology-network/common-lib/common"
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	"github.com/arcology-network/component-lib/storage"
	ccurl "github.com/arcology-network/concurrenturl/v2"
	urlcmn "github.com/arcology-network/concurrenturl/v2/common"
	curstorage "github.com/arcology-network/concurrenturl/v2/storage"
	"github.com/arcology-network/concurrenturl/v2/type/commutative"
	evmcmn "github.com/arcology-network/evm/common"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	adaptor "github.com/arcology-network/vm-adaptor/evm"
	"go.uber.org/zap"
)

type ExecutorResponse struct {
	Responses       []*cmntyp.ExecuteResponse
	Uuid            uint64
	SerialID        int
	Total           int
	Relation        map[ethcmn.Hash][]ethcmn.Hash
	SpawnHashes     map[ethcmn.Hash]ethcmn.Hash
	ContractAddress []ethcmn.Address
	Txids           map[ethcmn.Hash]*exetyp.DefItem
	CallResults     [][]byte
}

const (
	execStateWaitDBCommit = iota
	execStateWaitBlockStart
	execStateReady
	execStateProcessing
)

type pendingTask struct {
	precedings         []*ethcmn.Hash
	baseOn             *urlcmn.DatastoreInterface
	totalPrecedingSize int

	sequence  *cmntyp.ExecutingSequence
	timestamp *big.Int
	debug     bool
}

func (ps *pendingTask) match(results []*cmntyp.EuResult) bool {
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
	eus          []ExecutionImpl
	taskCh       chan *exetyp.ExecMessagers
	resultCh     chan *ExecutorResponse
	genExecLog   bool
	pendingTasks map[ethcmn.Hash][]*pendingTask
	url          *ccurl.ConcurrentUrl
	numTasks     int
	requestId    uint64
}

func NewExecutor(concurrency int, groupId string) actor.IWorkerEx {
	exec := &Executor{
		state:        execStateWaitDBCommit,
		height:       math.MaxUint64,
		snapshotDict: exetyp.NewLookup(),
		eus:          make([]ExecutionImpl, concurrency),
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
}

func (exec *Executor) OnStart() {
	// The following code were copied from exec v1.
	config := exetyp.MainConfig()
	for i := 0; i < int(exec.Concurrency); i++ {
		persistentDB := cachedstorage.NewDataStore()
		platform := urlcmn.NewPlatform()
		meta, _ := commutative.NewMeta(platform.Eth10Account())
		persistentDB.Inject(platform.Eth10Account(), meta)
		db := curstorage.NewTransientDB(persistentDB)

		url := ccurl.NewConcurrentUrl(db)
		api := adaptor.NewAPIV2(db, url)
		statedb := adaptor.NewStateDBV2(api, db, url)

		exec.eus[i] = &exetyp.ExecutionService{}
		exec.eus[i].Init(adaptor.NewEUV2(config.ChainConfig, *config.VMConfig, config.Chain, statedb, api, db, url), url)
	}
	exec.startExec()
}

func (exec *Executor) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch exec.state {
	case execStateWaitDBCommit:
		db := msg.Data.(*actor.CombinerElements).Get(actor.MsgApcHandle).Data.(*urlcmn.DatastoreInterface)
		exec.snapshotDict.Reset(db)
		exec.height = msg.Height + 1
		exec.state = execStateWaitBlockStart
	case execStateWaitBlockStart:
		combined := msg.Data.(*actor.CombinerElements)
		coinbase := evmcmn.BytesToAddress(combined.Get(actor.MsgBlockStart).Data.(*actor.BlockStart).Coinbase.Bytes())
		exec.execParams = &exetyp.ExecutorParameter{
			ParentInfo: combined.Get(actor.MsgParentInfo).Data.(*cmntyp.ParentInfo),
			Coinbase:   &coinbase,
			Height:     combined.Get(actor.MsgBlockStart).Height,
		}
		exec.state = execStateReady
	case execStateReady:
		switch msg.Name {
		case actor.MsgTxsToExecute:
			request := msg.Data.(*cmntyp.ExecutorRequest)
			exec.numTasks = len(request.Sequences)
			exec.requestId = msg.Msgid
			exec.pendingTasks = make(map[ethcmn.Hash][]*pendingTask)
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
		results := make([]*cmntyp.EuResult, len(data))
		for i, d := range data {
			results[i] = d.(*cmntyp.EuResult)
		}
		for hash, pt := range exec.pendingTasks {
			if pt[0].match(results) {
				// The following code were copied from exec v1.
				db := curstorage.NewTransientDB(*pt[0].baseOn)
				exec.url.Init(db)
				txIds, transitions := storage.GetTransitions(results)
				exec.url.Import(transitions)
				exec.url.PostImport()
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
	snapshot urlcmn.DatastoreInterface,
	timestamp *big.Int,
	sequence *cmntyp.ExecutingSequence,
	debug bool,
) {
	//db := curstorage.NewTransientDB(snapshot)
	config := exetyp.MainConfig()
	config.Coinbase = exec.execParams.Coinbase
	config.BlockNumber = new(big.Int).SetUint64(exec.height)
	config.Time = timestamp
	config.ParentHash = evmcmn.BytesToHash(exec.execParams.ParentInfo.ParentHash.Bytes())
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

func (exec *Executor) startExec() {
	for i := 0; i < int(exec.Concurrency); i++ {
		index := i
		go func(index int) {
			for {
				task := <-exec.taskCh
				exec.eus[index].SetDB(task.Snapshot)
				logid := exec.AddLog(log.LogLevel_Debug, "exec request", zap.Int("mgs", len(task.Sequence.Msgs)), zap.String("coinbase", fmt.Sprintf("%x", task.Config.Coinbase)), zap.Int("index", index))
				interLog := exec.GetLogger(logid)
				resp, err := exec.eus[index].Exec(task.Sequence, task.Config, interLog, exec.genExecLog)
				if err != nil {
					exec.AddLog(log.LogLevel_Error, "Exec err", zap.String("err", err.Error()))
					continue
				}
				exec.sendResult(resp, task, task.Debug)
			}
		}(index)
	}
}

func (exec *Executor) sendResult(response *exetyp.ExecutionResponse, messages *exetyp.ExecMessagers, debug bool) {
	// The following code were copied from exec v1.
	exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult", zap.Bool("debug", debug))
	counter := len(response.EuResults)
	if counter > 0 {
		if !debug {
			euresults := cmntyp.Euresults(response.EuResults)
			exec.MsgBroker.Send(actor.MsgEuResults, &euresults)
			exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult MsgEuResults", zap.Int("euresults", len(euresults)))
		}
		txsResults := make([]*cmntyp.ExecuteResponse, counter)

		worker := func(start, end, idx int, args ...interface{}) {
			results := args[0].([]interface{})[0].([]*cmntyp.EuResult)
			responses := args[0].([]interface{})[1].([]*cmntyp.ExecuteResponse)

			for i := start; i < end; i++ {
				result := results[i]
				var df *cmntyp.DeferCall
				if result.DC != nil {
					df = &cmntyp.DeferCall{
						DeferID:         result.DC.DeferID,
						ContractAddress: cmntyp.Address(result.DC.ContractAddress),
						Signature:       result.DC.Signature,
					}
				}
				responses[i] = &cmntyp.ExecuteResponse{
					DfCall:  df,
					Hash:    ethcmn.BytesToHash([]byte(result.H)),
					Status:  result.Status,
					GasUsed: result.GasUsed,
				}
			}
		}
		cmncmn.ParallelWorker(len(response.EuResults), exec.Concurrency, worker, response.EuResults, txsResults)

		responses := ExecutorResponse{
			Responses:       txsResults,
			Uuid:            messages.Uuid,
			Total:           messages.Total,
			SerialID:        messages.SerialID,
			Relation:        response.Relation,
			SpawnHashes:     response.SpawnHashes,
			ContractAddress: response.ContractAddress,
			Txids:           response.Txids,
		}
		if debug {
			responses.CallResults = response.CallResults
		} else {
			responses.CallResults = [][]byte{}
		}
		// exec.MsgBroker.Send(actor.MsgTxsExecuteResults, &responses)
		exec.resultCh <- &responses

		if !debug {
			tarss := cmntyp.TxAccessRecordSet(response.AccessRecords)
			exec.MsgBroker.Send(actor.MsgTxAccessRecords, &tarss)
			exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult MsgTxAccessRecords", zap.Int("MsgTxAccessRecords", len(tarss)))
		}
	}

	if !debug && len(response.ExecutingLogs) > 0 {
		exec.MsgBroker.Send(actor.MsgExecutingLogs, response.ExecutingLogs)
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult ExecutingLogs", zap.Int("ExecutingLogs", len(response.ExecutingLogs)))
	}

	if debug {
		return
	}
	exec.AddLog(log.LogLevel_Info, "send rws", zap.Int("nums", counter))
	counter = len(response.Receipts)
	if counter > 0 {
		exec.MsgBroker.Send(actor.MsgReceipts, &response.Receipts)
		receiptHashList := exec.toReceiptsHash(&response.Receipts)
		exec.MsgBroker.Send(actor.MsgReceiptHashList, receiptHashList)
		exec.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult MsgReceiptHashList", zap.Int("receiptHashList", len(receiptHashList.TxHashList)))
	}
	exec.AddLog(log.LogLevel_Info, "send receipt ", zap.Int("nums", counter))
}

func (e *Executor) toReceiptsHash(receipts *[]*ethtyp.Receipt) *cmntyp.ReceiptHashList {
	// The following code were copied from exec v1.
	rcptLength := 0
	if receipts != nil {
		rcptLength = len(*receipts)
	}

	if rcptLength == 0 {
		return &cmntyp.ReceiptHashList{}
	}

	receiptHashList := make([]ethcmn.Hash, rcptLength)
	txHashList := make([]ethcmn.Hash, rcptLength)
	gasUsedList := make([]uint64, rcptLength)
	worker := func(start, end, idx int, args ...interface{}) {
		receipts := args[0].([]interface{})[0].([]*ethtyp.Receipt)
		receiptHashList := args[0].([]interface{})[1].([]ethcmn.Hash)
		txHashList := args[0].([]interface{})[2].([]ethcmn.Hash)
		gasUsedList := args[0].([]interface{})[3].([]uint64)

		for i := range receipts[start:end] {
			idx := i + start
			receipt := receipts[idx]
			txHashList[idx] = receipt.TxHash
			receiptHashList[idx] = ethcmn.RlpHash(receipt)
			gasUsedList[idx] = receipt.GasUsed
		}
	}
	cmncmn.ParallelWorker(len(*receipts), e.Concurrency, worker, *receipts, receiptHashList, txHashList, gasUsedList)
	return &cmntyp.ReceiptHashList{
		TxHashList:      txHashList,
		ReceiptHashList: receiptHashList,
		GasUsedList:     gasUsedList,
	}
}
