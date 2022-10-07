package scheduler

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
	"github.com/HPISTechnologies/component-lib/log"
	"github.com/HPISTechnologies/main/modules/scheduler/lib"
	schedulingTypes "github.com/HPISTechnologies/main/modules/scheduler/types"
	"go.uber.org/zap"
)

type Scheduler struct {
	actor.WorkerThread
	schedule          *schedulingTypes.ExecutingSchedule
	dm                *DependencyManager
	exec              Executor
	parallelism       int
	parallelismTmp    int
	contractLookup    *schedulingTypes.ContractLookup
	schedulerInstance *lib.Scheduler
	generationIdx     int
	batchIdx          int
	conflictFile      string
}

var (
	schedulerSingleton actor.IWorkerEx
	initOnce           sync.Once
)

//return a Subscriber struct
func NewScheduler(concurrency int, groupid string) actor.IWorkerEx {
	initOnce.Do(func() {
		scheduler := Scheduler{}
		scheduler.Set(concurrency, groupid)
		scheduler.schedulerInstance, _ = lib.Start("")

		schedulerSingleton = &scheduler
	})

	return schedulerSingleton
}

func (schd *Scheduler) Inputs() ([]string, bool) {
	return []string{
		actor.MsgMessagersReaped,
		actor.MsgBlockStart,
	}, true
}

func (schd *Scheduler) Outputs() map[string]int {
	return map[string]int{
		actor.MsgInclusive:        1,
		actor.MsgExecTime:         1,
		actor.MsgSpawnedRelations: 1,
	}
}

func (schd *Scheduler) Config(params map[string]interface{}) {
	var executors []string
	for _, svc := range intf.Router.GetAvailableServices() {
		if strings.HasPrefix(svc, "executor") {
			executors = append(executors, svc)
		}
	}

	schd.parallelismTmp = int(params["parallelism"].(float64))
	schd.conflictFile = params["conflict_file"].(string)
	schd.dm = NewDependencyManager(
		NewRpcClientArbitrate(),
		schd.Concurrency,
	)
	schd.exec = NewRpcClientExec(
		executors,
		int(params["batch_size"].(float64)),
	)
}

func (sched *Scheduler) OnStart() {
	sched.dm.Start()
	sched.exec.Start()
	schedule := schedulingTypes.NewExecutingSchedule(sched.schedulerInstance)
	//sched.AddLog(log.LogLevel_Debug, "---------------------------Before NewExecutingSchedule", zap.String("errlog", errlog))
	sched.schedule = schedule
	sched.contractLookup = schedulingTypes.NewContractLookup()

	logstr := sched.schedule.Init(sched.conflictFile)
	sched.AddLog(log.LogLevel_Debug, "---------------------------OnStart update", zap.String("losg", logstr))
}

func (sched *Scheduler) Stop() {
	sched.dm.Stop()
	sched.exec.Stop()
}
func wraper(msgs []*types.StandardMessage) []*schedulingTypes.Message {
	schedulingMsgs := make([]*schedulingTypes.Message, len(msgs))
	for i := range msgs {
		msg := schedulingTypes.Message{
			Message:          msgs[i],
			IsSpawned:        false,
			DirectPrecedings: &[]*ethCommon.Hash{},
		}
		schedulingMsgs[i] = &msg
	}
	return schedulingMsgs
}
func (sched *Scheduler) GetBatch(schedulingMsgs []*schedulingTypes.Message, txids []uint32) (map[ethCommon.Hash]*schedulingTypes.Message, []*types.ExecutingSequence) {
	lookup := map[ethCommon.Hash]*schedulingTypes.Message{}
	var sequences []*types.ExecutingSequence
	if len(schedulingMsgs) == 0 {
		sched.generationIdx = sched.generationIdx + 1
		sched.batchIdx = 0

		sequences = sched.schedule.GetNextBatch()
		for _, sequence := range sequences {
			if sequence.Parallel {
				for _, msg := range sequence.Msgs {
					lookup[msg.TxHash] = &schedulingTypes.Message{
						Message:   msg,
						IsSpawned: false,
					}
				}
			} else {
				lookup[sequence.SequenceId] = &schedulingTypes.Message{
					Message: &types.StandardMessage{
						TxHash: sequence.SequenceId,
					},
					IsSpawned: false,
				}
			}
		}
	} else {
		msgs := make([]*types.StandardMessage, len(schedulingMsgs))
		for i, msg := range schedulingMsgs {
			lookup[msg.Message.TxHash] = msg
			msgs[i] = msg.Message
		}
		sequence := types.NewExecutingSequence(msgs, true)
		sequence.Txids = txids
		sequences = []*types.ExecutingSequence{sequence}
	}

	return lookup, sequences
}

func (sched *Scheduler) OnMessageArrived(msgs []*actor.Message) error {
	var timestamp *big.Int
	var messages []*types.StandardMessage
	height := uint64(0)
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgBlockStart:
			timestamp = v.Data.(*actor.BlockStart).Timestamp
		case actor.MsgMessagersReaped:
			sendingStandardMessages := v.Data.(types.SendingStandardMessages)
			messages = sendingStandardMessages.ToMessages()
			height = v.Height
		}
	}

	sched.dm.txAddressLookup.Reset(messages)

	sched.parallelism = sched.parallelismTmp
	sched.AddLog(log.LogLevel_Info, "Before NewExecutionSchedule", zap.Int("messages", len(messages)))

	//*************************for test***************************************
	checks := map[ethCommon.Address]int{}
	for _, msg := range messages {
		to := msg.Native.To()
		if to != nil {
			checks[*to] = 0
		}
	}
	sched.AddLog(log.LogLevel_Info, ">>>>>>>>>>>>>>>>>>>>>>", zap.Int("contractAddress", len(checks)))
	//***************************for test****************************************
	transfers, contracts := sched.contractLookup.Divide(messages)

	sched.AddLog(log.LogLevel_Info, ">>>>>>>>>>>>>>>>>>>>>>", zap.Int("transfers", len(transfers)), zap.Int("contracts", len(contracts)))
	starttime := time.Now()
	err, count, idStart := sched.schedule.Reset(contracts, height)
	if len(err) > 0 {
		sched.AddLog(log.LogLevel_Error, "schedule.Reset err", zap.String("err", err))
	}
	sched.AddLog(log.LogLevel_Info, "after schedule.Reset ------->", zap.Int("generations", count))

	schedulingMsgs := wraper(transfers)
	ids, _ := schedulingTypes.CreateIds(idStart, len(schedulingMsgs))
	if len(schedulingMsgs) > 0 {
		sched.generationIdx = 0 //transactions
	} else {
		sched.generationIdx = -1 //new generation
	}
	msgsToExecute, sequences := sched.GetBatch(schedulingMsgs, ids)

	for i, sequence := range sequences {
		sched.AddLog(log.LogLevel_Debug, "sequence type", zap.Int("sequence idx", i), zap.Bool("Parallel", sequence.Parallel), zap.Int("msgCount", len(sequence.Msgs)))
	}

	logid := sched.AddLog(log.LogLevel_Info, "Before first batch", zap.Int("txs", len(msgsToExecute)), zap.Int("sequences", len(sequences)))
	interLog := sched.GetLogger(logid)
	finished := sched.dm.OnNewBatch(msgsToExecute, sched.MsgBroker, interLog, sequences)
	for !finished {
		logid := sched.AddLog(log.LogLevel_Info, "start enter exec")
		interLog = sched.GetLogger(logid)
		results, spawnedHashes, relations, contractAddress, deftxids, defCallees := sched.exec.Run(msgsToExecute, sequences, timestamp, msgs[0], interLog, sched.parallelism, sched.generationIdx, sched.batchIdx)
		sched.contractLookup.UpdateContract(contractAddress)
		sched.dm.txAddressLookup.Append(defCallees)
		logid = sched.AddLog(log.LogLevel_Info, "Before DependencyManager.OnExecResultReceived")
		interLog := sched.GetLogger(logid)
		spawned, txids := sched.dm.OnExecResultReceived(results, spawnedHashes, relations, sched.MsgBroker, interLog, msgsToExecute, deftxids, sched.generationIdx, sched.batchIdx)
		if len(spawned) > 0 {
			sched.batchIdx = sched.batchIdx + 1
		}
		msgsToExecute, sequences = sched.GetBatch(spawned, txids)
		for i, sequence := range sequences {
			sched.AddLog(log.LogLevel_Debug, "sequence type", zap.Int("sequence idx", i), zap.Bool("Parallel", sequence.Parallel), zap.Int("msgCount", len(sequence.Msgs)))
		}
		logid = sched.AddLog(log.LogLevel_Info, "Before DependencyManager.OnNewBatch", zap.Int("len(msgsToExecute)", len(msgsToExecute)))
		interLog = sched.GetLogger(logid)
		finished = sched.dm.OnNewBatch(msgsToExecute, sched.MsgBroker, interLog, sequences)
	}
	execTime := time.Since(starttime)
	statisticInfo := types.StatisticalInformation{
		Key:      actor.MsgExecTime,
		TimeUsed: execTime,
		Value:    fmt.Sprintf("%v", execTime),
	}
	sched.MsgBroker.Send(actor.MsgExecTime, &statisticInfo)
	sched.dm.OnBlockCompleted()

	conflicts := sched.dm.conflictlist.GetAllPairs()
	for _, item := range conflicts {
		sched.schedulerInstance.Update([]*ethCommon.Address{item.Left}, []*ethCommon.Address{item.Right})
	}
	sched.dm.conflictlist.Clear()

	return nil
}

func (sched *Scheduler) SetParallelism(ctx context.Context, request *types.ClusterConfig, response *types.SetReply) error {
	sched.parallelismTmp = request.Parallelism
	return nil
}
