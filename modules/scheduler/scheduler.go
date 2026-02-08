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
	"encoding/json"

	"github.com/arcology-network/common-lib/common"
	types "github.com/arcology-network/common-lib/types"

	// engine "github.com/arcology-network/main/modules/scheduler/lib"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"

	eucommon "github.com/arcology-network/common-lib/types"
	schtyp "github.com/arcology-network/main/modules/scheduler/types"
	scheduler "github.com/arcology-network/scheduler"
	"github.com/arcology-network/storage-committer/type/univalue"

	scommon "github.com/arcology-network/streamer/common"
)

const (
	MaxBlockSize = 50000
)

const (
	scheduleStateInit = iota
	scheduleStateReady
	scheduleStateExec
	scheduleStateApc
	scheduleStateFeedback
)

type Scheduler struct {
	schdEngine   *scheduler.Scheduler
	conflictFile string

	// Data structures used in one block.
	context       *processContext
	transfers     []*eucommon.StandardMessage
	contractCalls []*eucommon.StandardMessage

	// Data structures used all the time.
	contractDict map[evmCommon.Address]struct{}

	state int

	schdState *mtypes.SchdState
}

func NewScheduler() actor.Business {
	schdEngine, _ := scheduler.NewScheduler("", false)
	return &Scheduler{
		schdEngine:    schdEngine,
		context:       createProcessContext(),
		transfers:     make([]*eucommon.StandardMessage, 0, MaxBlockSize),
		contractCalls: make([]*eucommon.StandardMessage, 0, MaxBlockSize),
		contractDict:  make(map[evmCommon.Address]struct{}),
		state:         scheduleStateInit,
	}
}

func (schd *Scheduler) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgInitScheduletate,
		actor.CombinedName(scommon.MsgMessagersReaped, scommon.MsgBlockStart),
		scommon.MsgFeedBacks,
		scommon.MsgExecGeneration,
		scommon.MsgApcHandle,
	}, false
}

func (schd *Scheduler) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgInclusive:                  1,
		scommon.MsgSchdState:                  1,
		scommon.MsgGenerationReapingList:      1,
		scommon.MsgGenerationReapingCompleted: 1,
		scommon.MsgExecGeneration:             1,
	}
}

func (schd *Scheduler) Config(params map[string]interface{}) {
	execBatchSize := params["batch_size"].(int)
	schd.conflictFile = params["conflict_file"].(string)
	jsonStr, _ := json.Marshal(params["executors"])
	confs := []*mtypes.ExecutorConf{}
	json.Unmarshal(jsonStr, &confs)

	schd.context.init(execBatchSize, confs)
}

func (schd *Scheduler) GetFSMRules() map[int]actor.FSMRule {
	return map[int]actor.FSMRule{
		scheduleStateInit: {Accept: []string{scommon.MsgInitScheduletate}},
		scheduleStateReady: {Accept: []string{
			actor.CombinedName(scommon.MsgMessagersReaped, scommon.MsgBlockStart),
		}},
		scheduleStateExec:     {Accept: []string{scommon.MsgExecGeneration}},
		scheduleStateApc:      {Accept: []string{scommon.MsgApcHandle}},
		scheduleStateFeedback: {Accept: []string{scommon.MsgFeedBacks}},
	}
}

func (schd *Scheduler) GetCurrentState() int {
	return schd.state
}

func (schd *Scheduler) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgInitScheduletate, schd.InitSchedule)
	reg.Register(actor.CombinedName(scommon.MsgMessagersReaped, scommon.MsgBlockStart), schd.startCreateGenerations)
	reg.Register(scommon.MsgExecGeneration, schd.startGenerationExec)
	reg.Register("onExecResult", schd.onExecResult)
	reg.Register("onArbResult", schd.onArbResult)
	reg.Register(scommon.MsgApcHandle, schd.waitingApc)
	reg.Register("afterSaveSchedule", schd.afterSaveSchedule)
	reg.Register(scommon.MsgFeedBacks, schd.feedback)
}

func (schd *Scheduler) InitSchedule(ctx *actor.ActionContext) error {
	schtyp.NewScheduleLoader(schd.schdEngine).Init(schd.conflictFile)
	states := ctx.Messages[0].Data.([]mtypes.SchdState)
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

	schd.ChangeState(ctx, scheduleStateReady, "scheduleStateReady")
	return nil
}

func (schd *Scheduler) startCreateGenerations(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	combined := msg.Data.(*actor.CombinerElements)
	schd.context.timestamp = combined.Get(scommon.MsgBlockStart).Data.(*actor.BlockStart).Timestamp
	ctx.ExecCtx.LogInfo("received messagersReaped")
	stdMsgs := combined.Get(scommon.MsgMessagersReaped).Data.([]*eucommon.StandardMessage)

	schd.context.onNewBlock(msg.Height)

	ctx.ExecCtx.LogInfo("start new schedule", logger.F("messages", len(stdMsgs)))
	schd.splitMessagesByType(stdMsgs)

	schd.context.onStartBlock(schd.createGenerations())

	ctx.ExecCtx.Send(scommon.MsgExecGeneration, "")

	schd.ChangeState(ctx, scheduleStateExec, "scheduleStateExec")
	return nil
}

func (schd *Scheduler) startGenerationExec(ctx *actor.ActionContext) error {
	if schd.context.generationCount == 0 {
		ctx.ExecCtx.Send(scommon.MsgGenerationReapingList, &types.InclusiveList{
			HashList:          []evmCommon.Hash{},
			Successful:        []bool{},
			NextGenerationIdx: 0,
		})

		schd.ChangeState(ctx, scheduleStateApc, "scheduleStateApc")
	} else {
		schd.context.onNewGeneration()
		schd.context.GetCurrentGeneration().startProcess(ctx.ExecCtx)
	}
	return nil
}
func (schd *Scheduler) onExecResult(ctx *actor.ActionContext) error {
	resp := ctx.Messages[0].Data.(*mtypes.ExecutorResponses)
	currentGeneration := schd.context.GetCurrentGeneration()
	if !currentGeneration.OnExecResult(resp) {
		ctx.ExecCtx.LogDebug("OnExecResult next", logger.F("list", schd.context.executed))
		currentGeneration.NextProcess(ctx.ExecCtx, int(resp.ExecId))
		return nil
	} else {
		ctx.ExecCtx.LogDebug("OnExecResult end", logger.F("list", schd.context.executed))
	}

	//start arbitrate
	currentGeneration.StartArbitrate(ctx.ExecCtx)
	return nil
}

func (schd *Scheduler) onArbResult(ctx *actor.ActionContext) error {
	resp := ctx.Messages[0].Data.(*mtypes.ArbitratorResponse)
	currentGen := schd.context.GetCurrentGeneration()

	currentGen.onArbitrateResult(ctx.ExecCtx, resp)
	list := currentGen.CollectGenerationResult()
	ctx.ExecCtx.LogDebug("onArbResult", logger.F("list", schd.context.executed))
	ctx.ExecCtx.LogDebug("MsgGenerationReapingList", logger.F("list", list))
	ctx.ExecCtx.Send(scommon.MsgGenerationReapingList, list, schd.context.height)

	schd.ChangeState(ctx, scheduleStateApc, "scheduleStateApc")
	return nil
}

func (schd *Scheduler) waitingApc(ctx *actor.ActionContext) error {
	if schd.context.generationCount > 0 && schd.context.currentGenerationID+1 < schd.context.generationCount {
		//next generation
		ctx.ExecCtx.Send(scommon.MsgExecGeneration, "")
		return nil
	}

	ctx.ExecCtx.Send(scommon.MsgGenerationReapingCompleted, 1)

	// Send summarized results.
	// State changes of Scheduler.
	conflictL, conflictR, conflictSL, conflictSR := schd.context.conflicts.Format()
	schd.schdState = &mtypes.SchdState{
		Height:                ctx.Messages[0].Height,
		NewContracts:          schd.context.newContracts,
		ConflictionLefts:      conflictL,
		ConflictionRights:     conflictR,
		ConflictionLeftSigns:  conflictSL,
		ConflictionRightSigns: conflictSR,
	}

	ctx.ExecCtx.InvokeRPC("schdstore", "Save", schd.schdState, "afterSaveSchedule")
	return nil
}

func (schd *Scheduler) afterSaveSchedule(ctx *actor.ActionContext) error {
	ctx.ExecCtx.Send(scommon.MsgSchdState, schd.schdState)
	// Inclusive list.
	flags := make([]bool, len(schd.context.executed))
	for i, hash := range schd.context.executed {
		if _, ok := schd.context.deletedDict[hash]; !ok {
			flags[i] = true
		}
	}
	ctx.ExecCtx.Send(scommon.MsgInclusive, &types.InclusiveList{
		HashList:          schd.context.executed,
		Successful:        flags,
		NextGenerationIdx: 0,
	})

	ctx.ExecCtx.LogInfo("send inclusive", logger.F("count", len(flags)))

	// Update states of scheduler.
	common.MergeMaps(schd.contractDict, common.SliceToDict(schd.context.newContracts))
	if len(schd.schdState.ConflictionLefts) > 0 {

		// Add all the conflicted addresses into contractDict,
		// since we may miss some contract deployments.
		common.MergeMaps(schd.contractDict, common.SliceToDict(schd.schdState.ConflictionLefts))
		common.MergeMaps(schd.contractDict, common.SliceToDict(schd.schdState.ConflictionRights))
	}
	for _, ci := range schd.context.conflicts.Conflicts {
		schd.schdEngine.Add(ci.LeftAddress, ci.LeftSign, ci.RightAddress, ci.RightSign)
	}

	schd.ChangeState(ctx, scheduleStateFeedback, "scheduleStateFeedback")
	return nil
}

func (schd *Scheduler) ChangeState(ctx *actor.ActionContext, state int, stateName string) error {
	schd.state = state
	ctx.ExecCtx.LogDebug("******business state change into " + stateName)
	return nil
}

func (schd *Scheduler) feedback(ctx *actor.ActionContext) error {
	univalues := ctx.Messages[0].Data.([]*univalue.Univalue)
	schd.schdEngine.Import(univalues)

	schd.ChangeState(ctx, scheduleStateReady, "scheduleStateReady")
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
	gens := ParseResult(schd.schdEngine.New(schd.contractCalls).Optimize(schd.schdEngine), len(schd.contractCalls))
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
