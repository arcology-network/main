package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/arcology-network/common-lib/common"
	types "github.com/arcology-network/common-lib/types"

	// engine "github.com/arcology-network/main/modules/scheduler/lib"

	"github.com/arcology-network/main/modules/storage"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"

	eucommon "github.com/arcology-network/eu/common"
	scheduler "github.com/arcology-network/eu/new-scheduler"
	schtyp "github.com/arcology-network/main/modules/scheduler/types"
)

type Scheduler struct {
	actor.WorkerThread

	schdEngine    *scheduler.Scheduler
	initOnce      sync.Once
	parallelism   int
	conflictFile  string
	execBatchSize int

	// Data structures used in one block.
	context       *processContext
	transfers     []*eucommon.StandardMessage
	contractCalls []*eucommon.StandardMessage

	// Data structures used all the time.
	contractDict map[evmCommon.Address]struct{}
}

var (
	schdInstance actor.IWorkerEx
	initSchdOnce sync.Once
)

func NewScheduler(concurrency int, groupId string) actor.IWorkerEx {
	initSchdOnce.Do(func() {
		schdEngine, _ := scheduler.NewScheduler("", false)
		schd := &Scheduler{
			schdEngine:    schdEngine,
			context:       createProcessContext(),
			transfers:     make([]*eucommon.StandardMessage, 0, 50000),
			contractCalls: make([]*eucommon.StandardMessage, 0, 50000),
			contractDict:  make(map[evmCommon.Address]struct{}),
		}
		schd.Set(concurrency, groupId)
		schdInstance = schd
	})
	return schdInstance
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
		actor.MsgSchdState:        1,
	}
}

func (schd *Scheduler) Config(params map[string]interface{}) {
	schd.execBatchSize = int(params["batch_size"].(float64))
	schd.parallelism = int(params["parallelism"].(float64))
	schd.conflictFile = params["conflict_file"].(string)
}

func (schd *Scheduler) OnStart() {}

func (schd *Scheduler) OnMessageArrived(msgs []*actor.Message) error {
	schd.initOnce.Do(func() {
		schd.context.init(schd.execBatchSize)
		schtyp.NewScheduleLoader(schd.schdEngine).Init(schd.conflictFile)

		var states []storage.SchdState
		var na int
		intf.Router.Call("schdstore", "Load", &na, &states)
		previous := uint64(0)
		for _, state := range states {
			if state.Height == previous {
				continue
			}

			previous = state.Height
			common.MergeMaps(schd.contractDict, common.SliceToDict(state.NewContracts))

			for i := range state.ConflictionLefts {
				schd.schdEngine.Add(state.ConflictionLefts[i], state.ConflictionLeftSigns[i], state.ConflictionRights[i], state.ConflictionRightSigns[i])
			}
		}
	})

	var stdMsgs []*eucommon.StandardMessage
	height := uint64(0)
	for _, msg := range msgs {
		switch msg.Name {
		case actor.MsgBlockStart:
			schd.context.timestamp = msg.Data.(*actor.BlockStart).Timestamp
		case actor.MsgMessagersReaped:
			schd.CheckPoint("received messagersReaped")
			height = msg.Height
			stdMsgs = msg.Data.([]*eucommon.StandardMessage)
			fmt.Printf("start new schedule height:%v\n", msg.Height)
		}
	}

	schd.CheckPoint("start new schedule", zap.Int("messages", len(stdMsgs)))
	schd.splitMessagesByType(stdMsgs)

	timeStart := time.Now()
	schd.context.onNewBlock(height)
	schd.context.msgTemplate = msgs[0]
	schd.context.logger = schd.GetLogger(schd.AddLog(log.LogLevel_Info, "Before first generation"))
	schd.context.parallelism = schd.parallelism
	for _, gen := range schd.createGenerations() {
		schd.context.onNewGeneration()
		gen.process()
	}

	// Send summarized results.
	// State changes of Scheduler.
	conflictL, conflictR, conflictSL, conflictSR := schd.context.conflicts.Format()
	schdState := &storage.SchdState{
		Height:                msgs[0].Height,
		NewContracts:          schd.context.newContracts,
		ConflictionLefts:      conflictL,
		ConflictionRights:     conflictR,
		ConflictionLeftSigns:  conflictSL,
		ConflictionRightSigns: conflictSR,
	}
	var na int
	intf.Router.Call("schdstore", "Save", schdState, &na)
	schd.MsgBroker.Send(actor.MsgSchdState, schdState)
	// Inclusive list.
	flags := make([]bool, len(schd.context.executed))
	for i, hash := range schd.context.executed {
		if _, ok := schd.context.deletedDict[hash]; !ok {
			flags[i] = true
		}
	}
	schd.MsgBroker.Send(actor.MsgInclusive, &types.InclusiveList{
		HashList:   schd.context.executed,
		Successful: flags,
	})
	schd.CheckPoint("send inclusive")
	// Spawned transactions.
	// schd.MsgBroker.Send(actor.MsgSpawnedRelations, schd.context.spawnedRelations)
	// Exec time.
	execTime := time.Since(timeStart)
	schd.MsgBroker.Send(actor.MsgExecTime, &mtypes.StatisticalInformation{
		Key:      actor.MsgExecTime,
		TimeUsed: execTime,
		Value:    fmt.Sprintf("%v", execTime),
	})

	// Update states of scheduler.
	common.MergeMaps(schd.contractDict, common.SliceToDict(schd.context.newContracts))
	if len(conflictL) > 0 {

		// Add all the conflicted addresses into contractDict,
		// since we may miss some contract deployments.
		common.MergeMaps(schd.contractDict, common.SliceToDict(conflictL))
		common.MergeMaps(schd.contractDict, common.SliceToDict(conflictR))
	}
	for _, ci := range schd.context.conflicts.Conflicts {
		schd.schdEngine.Add(ci.LeftAddress, ci.LeftSign, ci.RightAddress, ci.RightSign)
	}
	return nil
}

func (schd *Scheduler) SetParallelism(
	ctx context.Context,
	request *mtypes.ClusterConfig,
	response *mtypes.SetReply,
) error {
	schd.parallelism = request.Parallelism
	return nil
}

func (schd *Scheduler) splitMessagesByType(msgs []*eucommon.StandardMessage) {
	schd.transfers = schd.transfers[:0]
	schd.contractCalls = schd.contractCalls[:0]

	for _, msg := range msgs {
		if msg.Native.To == nil {
			schd.transfers = append(schd.transfers, msg)
			continue
		}

		if _, ok := schd.contractDict[*msg.Native.To]; ok {
			schd.contractCalls = append(schd.contractCalls, msg)
		} else {
			schd.transfers = append(schd.transfers, msg)
		}
	}
}

func (schd *Scheduler) createGenerations() []*generation {
	gens := ParseResult(schd.schdEngine.New(schd.contractCalls).Optimize(), len(schd.contractCalls))
	res := make([]*generation, 0, len(gens)+1)
	if len(schd.transfers) > 0 {
		res = append(res, newGeneration(
			schd.context,

			[]*mtypes.ExecutingSequence{mtypes.NewExecutingSequence(schd.transfers, true)},
		))
	}
	for _, gen := range gens {
		res = append(res, newGeneration(schd.context, gen))
	}
	return res
}

func ParseResult(scheduleList [][][]*eucommon.StandardMessage, msgsSize int) [][]*mtypes.ExecutingSequence {
	sequences := make([][]*mtypes.ExecutingSequence, len(scheduleList))
	for i, list := range scheduleList {
		if len(list) == 0 {
			continue
		}
		executingSequenceList := make([]*mtypes.ExecutingSequence, 0, len(list))
		parallels := make([]*eucommon.StandardMessage, 0, msgsSize)
		for _, msgs := range list {
			if len(msgs) == 0 {
				continue
			}
			if len(msgs) == 1 {
				parallels = append(parallels, msgs[0])
				continue
			}
			executingSequenceList = append(executingSequenceList, mtypes.NewExecutingSequence(msgs, false))
		}
		if len(parallels) > 0 {
			executingSequenceList = append(executingSequenceList, mtypes.NewExecutingSequence(parallels, true))
		}
		sequences[i] = executingSequenceList
	}
	return sequences
}
