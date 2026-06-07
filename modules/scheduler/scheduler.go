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
	"encoding/json"
	"fmt"

	types "github.com/arcology-network/common-lib/types"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"

	eucommon "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/scheduler/scheduler"

	cmap "github.com/arcology-network/common-lib/exp/map"
	profile "github.com/arcology-network/scheduler/callee"
	"github.com/arcology-network/scheduler/conflictor"
	scommon "github.com/arcology-network/streamer/common"
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
}

func NewScheduler() actor.Business {
	return &Scheduler{
		context:       createProcessContext(),
		transfers:     make([]*eucommon.StandardMessage, 0, mtypes.MaxBlockSize),
		contractCalls: make([]*eucommon.StandardMessage, 0, mtypes.MaxBlockSize),
		contractDict:  make(map[evmCommon.Address]struct{}),
		state:         scheduleStateInit,
	}
}

func (schd *Scheduler) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgInitialization,
		actor.CombinedName(scommon.MsgMessagersReaped, scommon.MsgBlockStart),
		scommon.MsgExecGeneration,
		scommon.MsgApcHandle,
	}, false
}

func (schd *Scheduler) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgInclusive:                  1,
		scommon.MsgGenerationReapingList:      1,
		scommon.MsgGenerationReapingCompleted: 1,
		scommon.MsgExecGeneration:             1,
	}
}

func (schd *Scheduler) Config(params map[string]interface{}) {
	execBatchSize := params["batch_size"].(int)
	jsonStr, _ := json.Marshal(params["executors"])
	confs := []*mtypes.ExecutorConf{}
	json.Unmarshal(jsonStr, &confs)

	schd.context.init(execBatchSize, confs)
}

func (schd *Scheduler) GetFSMRules() map[int]actor.FSMRule {
	return map[int]actor.FSMRule{
		scheduleStateInit: {Accept: []string{scommon.MsgInitialization}},
		scheduleStateReady: {Accept: []string{
			actor.CombinedName(scommon.MsgMessagersReaped, scommon.MsgBlockStart),
		}},
		scheduleStateExec: {Accept: []string{
			scommon.MsgExecGeneration,
		}},
		scheduleStateApc: {Accept: []string{scommon.MsgApcHandle}},
	}
}

func (schd *Scheduler) GetCurrentState() int {
	return schd.state
}

func (schd *Scheduler) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgInitialization, schd.InitSchedule)
	reg.Register(actor.CombinedName(scommon.MsgMessagersReaped, scommon.MsgBlockStart), schd.startCreateGenerations)
	reg.Register(scommon.MsgExecGeneration, schd.startGenerationExec)
	reg.Register("onExecResult", schd.onExecResult)
	reg.Register("onArbResult", schd.onArbResult)
	reg.Register(scommon.MsgApcHandle, schd.waitingApc)
	reg.Register("afterSaveSchedule", schd.afterSaveSchedule)
}

func (schd *Scheduler) InitSchedule(ctx *actor.ActionContext) error {
	store := ctx.Messages[0].Data.(*mtypes.Initialization).Store
	scheduler, err := scheduler.NewScheduler(profile.NewProfileManager(store, 1024))
	if err != nil {
		panic(err)
	}
	schd.schdEngine = scheduler

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

	schd.context.onStartBlock(schd.createGenerations(stdMsgs))

	ctx.ExecCtx.LogDebug("block scheduler", logger.F("transfer", len(schd.transfers)), logger.F("contracts", len(schd.contractCalls)), logger.F("generationCount", schd.context.generationCount))

	schd.ChangeState(ctx, scheduleStateExec, "scheduleStateExec")
	ctx.ExecCtx.Send(scommon.MsgExecGeneration, "")

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
	resp := ctx.Messages[0].Data.(*mtypes.ExecResponses)
	currentGeneration := schd.context.GetCurrentGeneration()
	if !currentGeneration.OnExecResult(resp) {
		currentGeneration.NextProcess(ctx.ExecCtx, int(resp.ExecId))
		return nil
	} else {
		ctx.ExecCtx.LogDebug("OnExecResult end")
	}
	//start arbitrate
	currentGeneration.StartArbitrate(ctx.ExecCtx)
	return nil
}

func (schd *Scheduler) onArbResult(ctx *actor.ActionContext) error {
	collisionSummary := ctx.Messages[0].Data.(*conflictor.CollisionSummary)
	ctx.ExecCtx.LogDebug("***** onArbResult *******", logger.F("collisionSummary", len(collisionSummary.Collisions)))
	for i := range collisionSummary.Collisions {
		fmt.Printf("***** Collisions idx: %v\n", i)
		collisionSummary.Collisions[i].Print()
	}
	currentGen := schd.context.GetCurrentGeneration()

	currentGen.onArbitrateResult(ctx.ExecCtx, collisionSummary)
	collisionSummary.MarkRollbackJobs(currentGen.gen)
	if !collisionSummary.IsEmpty() {
		scheduler.DebugPrecommit(schd.schdEngine, collisionSummary)
		scheduler.DebugCommit(schd.schdEngine)
	}
	list := currentGen.CollectGenerationResult()

	ctx.ExecCtx.Send(scommon.MsgGenerationReapingList, list, schd.context.height)
	schd.ChangeState(ctx, scheduleStateApc, "scheduleStateApc")
	return nil
}

func (schd *Scheduler) waitingApc(ctx *actor.ActionContext) error {
	if schd.context.generationCount > 0 && schd.context.currentGenerationID+1 < schd.context.generationCount {
		//next generation
		ctx.ExecCtx.Send(scommon.MsgExecGeneration, "")
		schd.ChangeState(ctx, scheduleStateExec, "scheduleStateExec")
		return nil
	}

	ctx.ExecCtx.Send(scommon.MsgGenerationReapingCompleted, 1)

	schd.afterSaveSchedule(ctx)
	return nil
}

func (schd *Scheduler) afterSaveSchedule(ctx *actor.ActionContext) error {
	// Inclusive list.
	failed := 0
	flags := make([]bool, len(schd.context.executed))
	for i, hash := range schd.context.executed {
		if _, ok := schd.context.deletedDict[hash]; !ok {
			flags[i] = true
		} else {
			failed++
		}
	}
	ctx.ExecCtx.Send(scommon.MsgInclusive, &types.InclusiveList{
		HashList:          schd.context.executed,
		Successful:        flags,
		NextGenerationIdx: 0,
	})

	ctx.ExecCtx.LogInfo("send inclusive", logger.F("count", len(flags)), logger.F("failed", failed), logger.F("newContract", len(schd.context.newContracts)))

	// Update states of scheduler.
	cmap.Merge(schd.contractDict, cmap.FromSlice(schd.context.newContracts, func(v evmCommon.Address) struct{} { return struct{}{} }))

	schd.ChangeState(ctx, scheduleStateReady, "scheduleStateReady")
	return nil
}

func (schd *Scheduler) ChangeState(ctx *actor.ActionContext, state int, stateName string) error {
	schd.state = state
	ctx.ExecCtx.LogDebug("****** " + ctx.ExecCtx.WorkCtx.BusinassName + "  state change into " + stateName)
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

func (schd *Scheduler) createGenerations(stdmsgs []*eucommon.StandardMessage) []*generation {
	eplan, err := schd.schdEngine.New(stdmsgs)
	if err != nil {
		logger.Log.Error(context.Background(), "scheduler", "createGenerations err", logger.F("err", err))
		return []*generation{}
	}
	res := make([]*generation, 0, len(eplan.Generations))
	for _, gen := range eplan.Generations {
		res = append(res, newGeneration(schd.context, gen))
	}
	return res
}
