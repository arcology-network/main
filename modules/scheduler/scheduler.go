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

package scheduler

import (
	"context"
	"sync"

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

	generationApcHandlerCh chan int
}

var (
	schdInstance actor.IWorkerEx
	initSchdOnce sync.Once
)

func NewScheduler(concurrency int, groupId string) actor.IWorkerEx {
	initSchdOnce.Do(func() {
		schdEngine, _ := scheduler.NewScheduler("", false)
		schd := &Scheduler{
			schdEngine:             schdEngine,
			context:                createProcessContext(),
			transfers:              make([]*eucommon.StandardMessage, 0, 50000),
			contractCalls:          make([]*eucommon.StandardMessage, 0, 50000),
			contractDict:           make(map[evmCommon.Address]struct{}),
			generationApcHandlerCh: make(chan int),
		}
		schd.Set(concurrency, groupId)
		schdInstance = schd
	})
	return schdInstance
}

func (schd *Scheduler) Inputs() ([]string, bool) {
	return []string{
		actor.CombinedName(actor.MsgMessagersReaped, actor.MsgBlockStart),
		actor.MsgApcHandle,
	}, false
}

func (schd *Scheduler) Outputs() map[string]int {
	return map[string]int{
		actor.MsgInclusive:                  1,
		actor.MsgExecTime:                   1,
		actor.MsgSpawnedRelations:           1,
		actor.MsgSchdState:                  1,
		actor.MsgGenerationReapingList:      1,
		actor.MsgGenerationReapingCompleted: 1,
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
	for _, msg := range msgs {
		switch msg.Name {
		case actor.CombinedName(actor.MsgMessagersReaped, actor.MsgBlockStart):
			combined := msg.Data.(*actor.CombinerElements)
			schd.context.timestamp = combined.Get(actor.MsgBlockStart).Data.(*actor.BlockStart).Timestamp
			schd.CheckPoint("received messagersReaped")
			stdMsgs = combined.Get(actor.MsgMessagersReaped).Data.([]*eucommon.StandardMessage)
			schd.ProcessMsgs(msgs[0], stdMsgs, msg.Height)
		}
	}
	return nil
}

func (schd *Scheduler) waitingApcHandler() {
	schd.CheckPoint("Waiting apchandler .......")
	<-schd.generationApcHandlerCh
	schd.CheckPoint("Got apchandler")
}

func (schd *Scheduler) ProcessMsgs(msg *actor.Message, stdMsgs []*eucommon.StandardMessage, height uint64) error {
	schd.CheckPoint("start new schedule", zap.Int("messages", len(stdMsgs)))
	schd.splitMessagesByType(stdMsgs)

	schd.context.onNewBlock(height)
	schd.context.msgTemplate = msg
	schd.context.logger = schd.GetLogger(schd.AddLog(log.LogLevel_Info, "Before first generation"))
	schd.context.parallelism = schd.parallelism
	gens := schd.createGenerations()
	for i, gen := range gens {
		schd.context.onNewGeneration()
		generationList := gen.process()
		generationIdx := i + 1 //for next generation execute
		if generationIdx == len(gens) {
			generationIdx = 0 //for next height
		}
		generationList.GenerationIdx = uint32(generationIdx)
		schd.MsgBroker.Send(actor.MsgGenerationReapingList, generationList)
		schd.waitingApcHandler()
	}
	if len(gens) == 0 {
		//this is a empty block
		schd.MsgBroker.Send(actor.MsgGenerationReapingList, &types.InclusiveList{
			HashList:      []evmCommon.Hash{},
			Successful:    []bool{},
			GenerationIdx: 0,
		})
		schd.waitingApcHandler()
	}

	schd.MsgBroker.Send(actor.MsgGenerationReapingCompleted, 1)

	// Send summarized results.
	// State changes of Scheduler.
	conflictL, conflictR, conflictSL, conflictSR := schd.context.conflicts.Format()
	schdState := &storage.SchdState{
		Height:                height,
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
		HashList:      schd.context.executed,
		Successful:    flags,
		GenerationIdx: 0,
	})
	schd.CheckPoint("send inclusive")

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

func (schd *Scheduler) NotifyApchandler(
	ctx context.Context,
	request uint64,
	response *int,
) error {
	if request == 0 {
		return nil
	}
	schd.generationApcHandlerCh <- 1
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
