package exec

import (
	"fmt"
	"math/big"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	ethTypes "github.com/HPISTechnologies/3rd-party/eth/types"
	"github.com/HPISTechnologies/common-lib/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/component-lib/log"
	ccurl "github.com/HPISTechnologies/concurrenturl/v2"
	urlcommon "github.com/HPISTechnologies/concurrenturl/v2/common"
	"github.com/HPISTechnologies/concurrenturl/v2/type/commutative"
	mevmCommon "github.com/HPISTechnologies/evm/common"
	execTypes "github.com/HPISTechnologies/main/modules/exec/types"
	adaptor "github.com/HPISTechnologies/vm-adaptor/evm"
	"go.uber.org/zap"

	cachedstorage "github.com/HPISTechnologies/common-lib/cachedstorage"
	curstorage "github.com/HPISTechnologies/concurrenturl/v2/storage"
)

type Executor struct {
	actor.WorkerThread
	nthread            int64
	euServices         []execTypes.ExecutionService
	config             *adaptor.Config
	executorParameters *execTypes.ExecutorParameter
	msgsChan           chan *execTypes.ExecMessagers
	exitChan           chan bool
	txsCache           map[uint64]*execTypes.TxToExecutes
	execlog            bool
}

//return a Subscriber struct
func NewExecutor(concurrency int, groupid string) actor.IWorkerEx {
	e := Executor{
		nthread:            int64(concurrency),
		euServices:         make([]execTypes.ExecutionService, concurrency),
		executorParameters: &execTypes.ExecutorParameter{},
		msgsChan:           make(chan *execTypes.ExecMessagers, 50),
		exitChan:           make(chan bool, 0),
		txsCache:           map[uint64]*execTypes.TxToExecutes{},
	}
	e.Set(concurrency, groupid)
	return &e
}

func (e *Executor) Inputs() ([]string, bool) {
	return []string{actor.MsgTxsToExecute, actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo)}, false
}

func (e *Executor) Outputs() map[string]int {
	return map[string]int{
		actor.MsgReceipts:          100,
		actor.MsgReceiptHashList:   100,
		actor.MsgEuResults:         100,
		actor.MsgTxAccessRecords:   100,
		actor.MsgTxsExecuteResults: 1,
		actor.MsgExecutingLogs:     1,
	}
}

func (e *Executor) OnStart() {
	e.config = execTypes.MainConfig()
	for i := 0; i < int(e.nthread); i++ {
		persistentDB := cachedstorage.NewDataStore()
		platform := urlcommon.NewPlatform()
		meta, _ := commutative.NewMeta(platform.Eth10Account())
		persistentDB.Inject(platform.Eth10Account(), meta)
		db := curstorage.NewTransientDB(persistentDB)

		url := ccurl.NewConcurrentUrl(db)
		api := adaptor.NewAPIV2(db, url)
		statedb := adaptor.NewStateDBV2(api, db, url)

		e.euServices[i].Eu = adaptor.NewEUV2(e.config.ChainConfig, *e.config.VMConfig, e.config.Chain, statedb, api, db, url)
		e.euServices[i].TxsQueue = execTypes.NewQueue()
		e.euServices[i].Url = url
	}
	e.startExec()
}

func (e *Executor) Config(params map[string]interface{}) {
	if v, ok := params["gatherexeclog"]; !ok {
		panic("parameter not found: gatherexeclog")
	} else {
		e.execlog = v.(bool)
	}
}

func (e *Executor) startExec() {
	for i := 0; i < int(e.nthread); i++ {
		index := i
		go func(index int) {
			for {
				select {
				case msgs := <-e.msgsChan:
					e.euServices[index].DB = msgs.Snapshot
					logid := e.AddLog(log.LogLevel_Debug, "exec request", zap.Int("mgs", len(msgs.Sequence.Msgs)), zap.String("coinbase", fmt.Sprintf("%x", msgs.Config.Coinbase)), zap.Int("index", index))
					interLog := e.GetLogger(logid)
					resp, err := e.euServices[index].Exec(msgs.Sequence, msgs.Config, interLog, e.execlog)
					if err != nil {
						e.AddLog(log.LogLevel_Error, "Exec err", zap.String("err", err.Error()))
						continue
					}
					e.sendResult(resp, msgs, msgs.Debug)
				case <-e.exitChan:
					break
				}
			}
		}(index)
	}
}

func (e *Executor) Stop() {

}
func (e *Executor) ExecMsgs(msgs *execTypes.TxToExecutes, height uint64) {
	if msgs == nil {
		return
	}
	e.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>> start exec request", zap.Int("e.Msgs", len(msgs.Sequences)), zap.Uint64("Parallelism", msgs.Parallelism))
	groups := e.Groups(msgs.Sequences, msgs.Parallelism)
	for i := range groups {
		config := execTypes.MainConfig()
		config.Coinbase = e.executorParameters.Coinbase
		config.BlockNumber = new(big.Int).SetUint64(height)
		config.Time = msgs.Timestamp
		config.ParentHash = mevmCommon.BytesToHash(e.executorParameters.ParentInfo.ParentHash.Bytes())
		groups[i].Snapshot = msgs.SnapShots
		groups[i].Config = config
		groups[i].Debug = msgs.Debug

		e.msgsChan <- groups[i]
	}
}
func (e *Executor) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo):
			combined := v.Data.(*actor.CombinerElements)
			coinbase := mevmCommon.BytesToAddress(combined.Get(actor.MsgBlockStart).Data.(*actor.BlockStart).Coinbase.Bytes())
			e.executorParameters = &execTypes.ExecutorParameter{
				ParentInfo: combined.Get(actor.MsgParentInfo).Data.(*types.ParentInfo),
				Coinbase:   &coinbase,
				Height:     combined.Get(actor.MsgBlockStart).Height,
			}
			if txs, ok := e.txsCache[e.executorParameters.Height]; ok {
				e.ExecMsgs(txs, e.executorParameters.Height)
				e.txsCache = map[uint64]*execTypes.TxToExecutes{}
			}
		case actor.MsgTxsToExecute:
			txs := msgs[0].Data.(*execTypes.TxToExecutes)
			if e.executorParameters.Height == v.Height {
				e.ExecMsgs(txs, v.Height)
			} else {
				e.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>> MsgTxsToExecute cached", zap.Uint64("e.executorParameters.Height", e.executorParameters.Height), zap.Uint64("v.Height", v.Height))
				e.txsCache[v.Height] = txs
			}
		}
	}
	return nil
}

func (e *Executor) Groups(sequences []*types.ExecutingSequence, parallelism uint64) []*execTypes.ExecMessagers {

	groups := make([]*execTypes.ExecMessagers, len(sequences))
	uuid := common.GenerateUUID()
	for i, sequence := range sequences {
		groups[i] = &execTypes.ExecMessagers{
			Uuid:     uuid,
			Total:    len(sequences),
			SerialID: i,
			Sequence: sequence,
		}
	}
	return groups
}

func (e *Executor) ToReceiptsHash(receipts *[]*ethTypes.Receipt) *types.ReceiptHashList {
	rcptLength := 0
	if receipts != nil {
		rcptLength = len(*receipts)
	}

	if rcptLength == 0 {
		return &types.ReceiptHashList{}
	}

	receiptHashList := make([]ethCommon.Hash, rcptLength)
	txHashList := make([]ethCommon.Hash, rcptLength)
	gasUsedList := make([]uint64, rcptLength)
	worker := func(start, end, idx int, args ...interface{}) {
		receipts := args[0].([]interface{})[0].([]*ethTypes.Receipt)
		receiptHashList := args[0].([]interface{})[1].([]ethCommon.Hash)
		txHashList := args[0].([]interface{})[2].([]ethCommon.Hash)
		gasUsedList := args[0].([]interface{})[3].([]uint64)

		for i := range receipts[start:end] {
			idx := i + start
			receipt := receipts[idx]
			txHashList[idx] = receipt.TxHash
			receiptHashList[idx] = ethCommon.RlpHash(receipt)
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

func (e *Executor) sendResult(response *execTypes.ExecutionResponse, messages *execTypes.ExecMessagers, debug bool) {
	e.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult", zap.Bool("debug", debug))
	counter := len(response.EuResults)
	if counter > 0 {
		if !debug {
			euresults := types.Euresults(response.EuResults)
			e.MsgBroker.Send(actor.MsgEuResults, &euresults)
			e.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult MsgEuResults", zap.Int("euresults", len(euresults)))
		}
		txsResults := make([]*types.ExecuteResponse, counter)

		worker := func(start, end, idx int, args ...interface{}) {
			results := args[0].([]interface{})[0].([]*types.EuResult)
			responses := args[0].([]interface{})[1].([]*types.ExecuteResponse)

			for i := start; i < end; i++ {
				result := results[i]
				var df *types.DeferCall
				if result.DC != nil {
					df = &types.DeferCall{
						DeferID:         result.DC.DeferID,
						ContractAddress: types.Address(result.DC.ContractAddress),
						Signature:       result.DC.Signature,
					}
				}
				responses[i] = &types.ExecuteResponse{
					DfCall:  df,
					Hash:    ethCommon.BytesToHash([]byte(result.H)),
					Status:  result.Status,
					GasUsed: result.GasUsed,
				}
			}
		}
		common.ParallelWorker(len(response.EuResults), e.Concurrency, worker, response.EuResults, txsResults)

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
		e.MsgBroker.Send(actor.MsgTxsExecuteResults, &responses)

		if !debug {
			tarss := types.TxAccessRecordSet(response.AccessRecords)
			e.MsgBroker.Send(actor.MsgTxAccessRecords, &tarss)
			e.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult MsgTxAccessRecords", zap.Int("MsgTxAccessRecords", len(tarss)))
		}

	}

	if !debug && len(response.ExecutingLogs) > 0 {
		e.MsgBroker.Send(actor.MsgExecutingLogs, response.ExecutingLogs)
		e.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult ExecutingLogs", zap.Int("ExecutingLogs", len(response.ExecutingLogs)))
	}

	if debug {
		return
	}
	e.AddLog(log.LogLevel_Info, "send rws", zap.Int("nums", counter))
	counter = len(response.Receipts)
	if counter > 0 {
		e.MsgBroker.Send(actor.MsgReceipts, &response.Receipts)
		receiptHashList := e.ToReceiptsHash(&response.Receipts)
		e.MsgBroker.Send(actor.MsgReceiptHashList, receiptHashList)
		e.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>>>>>>sendResult MsgReceiptHashList", zap.Int("receiptHashList", len(receiptHashList.TxHashList)))
	}
	e.AddLog(log.LogLevel_Info, "send receipt ", zap.Int("nums", counter))
}
