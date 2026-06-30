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

package storage

import (
	"math"

	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/ethereum/go-ethereum/triedb/hashdb"

	eushared "github.com/arcology-network/eu/shared"

	statecell "github.com/arcology-network/common-lib/crdt/statecell"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/state-engine/storage/proxy"

	statecache "github.com/arcology-network/state-engine/state/cache"
	statecommitter "github.com/arcology-network/state-engine/state/committer"
)

type DBOperation interface {
	Init(stateStore *statecache.ExecutionStateStore)
	InitAsync(ctx *actor.ActionContext)
	Import(transitions []*statecell.StateCell)
	PreCommit(ctx *actor.ActionContext, euResults []*eushared.EuResult, height uint64)
	PreCommitCompleted(ctx *actor.ExecutionContext, height uint64) [32]byte
	Commit(ctx *actor.ActionContext, height uint64)
	Outputs() map[string]int
	Config(params map[string]interface{})

	PreCommitAsync(ctx *actor.ExecutionContext)
	CommitAsync(ctx *actor.ExecutionContext, height uint64)

	//for rpc async callback
	AddMetas(ctx *actor.ActionContext)
	sendAsyncUrlUpdate(ctx *actor.ActionContext)
}

type BasicDBOperation struct {
	StateStore *statecache.ExecutionStateStore
	Committer  *statecommitter.StateCommitter

	Keys     []string
	Values   []interface{}
	AcctRoot [32]byte
}

func (op *BasicDBOperation) Init(stateStore *statecache.ExecutionStateStore) {
	op.StateStore = stateStore
	op.Committer = statecommitter.NewStateCommitter(op.StateStore.CommittedStore(), op.StateStore.GetWriters())
	op.Keys = []string{}
	op.Values = []interface{}{}
	op.AcctRoot = [32]byte{}
}

func (op *BasicDBOperation) Import(transitions []*statecell.StateCell) {
	op.Committer.Import(transitions)
}

func (op *BasicDBOperation) PreCommit(euResults []*eushared.EuResult, height uint64) {
	op.Committer.Finalize(GetTransitionIds(euResults))
	op.Committer.SyncPrecommit()
	op.Keys = []string{}
	op.Values = []interface{}{}
}

func (op *BasicDBOperation) PreCommitCompleted(ctx *actor.ExecutionContext) [32]byte {
	return op.StateStore.CommittedStore().(*proxy.StorageProxy).EthStore().Root()
}
func (op *BasicDBOperation) InitAsync() {

}

func (op *BasicDBOperation) PreCommitAsync(ctx *actor.ExecutionContext) {
	ctx.LogDebug("Before PreCommit Async.")
	op.Committer.AsyncPrecommit()
	ctx.LogDebug("After PreCommit Async.")
}
func (op *BasicDBOperation) CommitAsync(ctx *actor.ExecutionContext, height uint64) {
	ctx.LogDebug("Before Commit Async.")
	op.Committer.AsyncCommit(height)
	ctx.LogDebug("After Commit Async.")
}

func (op *BasicDBOperation) Commit(height uint64) {
	op.Committer.SyncCommit(height)
}

func (op *BasicDBOperation) Outputs() map[string]int {
	return map[string]int{}
}

func (op *BasicDBOperation) Config(params map[string]interface{}) {}

type CommitTask struct {
	Msg *scommon.Message
	Ctx *actor.ExecutionContext
}

const (
	dbStateUninit = iota
	dbStateInit
	dbStateDone
)

type DBHandler struct {
	StateStore             *statecache.ExecutionStateStore
	state                  int
	importMsg              string
	commitMsg              string
	generationCompletedMsg string
	finalizeMsg            string
	op                     DBOperation

	initDb          bool
	saveConfliction bool
	taskCh          chan *CommitTask
}

func NewDBHandler(importMsg, commitMsg, generationCompletedMsg, finalizeMsg string, op DBOperation, saveConflictions bool) *DBHandler {
	handler := &DBHandler{
		state:                  dbStateUninit,
		importMsg:              importMsg,
		commitMsg:              commitMsg,
		generationCompletedMsg: generationCompletedMsg,
		finalizeMsg:            finalizeMsg,
		op:                     op,
		initDb:                 false,
		taskCh:                 make(chan *CommitTask, 20),
		saveConfliction:        saveConflictions,
	}
	return handler
}

func (handler *DBHandler) Inputs() ([]string, bool) {
	msgs := []string{handler.importMsg, handler.generationCompletedMsg, handler.finalizeMsg}
	if handler.state == dbStateUninit {
		msgs = append(msgs, scommon.MsgInitialization)
	}
	msgs = append(msgs, actor.CombinedName(handler.commitMsg, scommon.MsgConflictTransitions))
	return msgs, false
}

func (handler *DBHandler) Outputs() map[string]int {
	outputs := handler.op.Outputs()
	return outputs
}

func (handler *DBHandler) Config(params map[string]interface{}) {
	dbpath := ""
	if v, ok := params["dbpath"]; !ok {
		panic("parameter not found: dbpath")
	} else {
		dbpath = v.(string)
	}
	if v, ok := params["init_db"]; !ok {
		panic("parameter not found: init_db")
	} else {
		if !v.(bool) {

			handler.StateStore = statecache.NewDefaultExecutionStateStore(proxy.NewPebbleDBProxy(dbpath, dbpath, math.MaxUint64, &hashdb.Config{CleanCacheSize: 1024 * 1024 * 100}))

			handler.op.Init(handler.StateStore)
			handler.initDb = true
		}
	}
	handler.op.Config(params)

	go func() {
		for {
			task := <-handler.taskCh

			switch task.Msg.Name {
			case actor.CombinedName(handler.commitMsg, scommon.MsgConflictTransitions):
				handler.op.PreCommitAsync(task.Ctx)
			case handler.generationCompletedMsg:
				handler.op.PreCommitCompleted(task.Ctx, task.Msg.Height)
			case handler.finalizeMsg:
				handler.op.CommitAsync(task.Ctx, task.Msg.Height)
			}

		}
	}()
}

func (handler *DBHandler) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgInitialization, handler.Initialization)
	reg.Register(handler.importMsg, handler.importData)
	reg.Register(actor.CombinedName(handler.commitMsg, scommon.MsgConflictTransitions), handler.startCommit)
	reg.Register(handler.generationCompletedMsg, handler.generationComplete)
	reg.Register(handler.finalizeMsg, handler.finalize)
	reg.Register("AddMetas", handler.AddMetas)
	reg.Register("sendAsyncUrlUpdate", handler.sendAsyncUrlUpdate)
}
func (handler *DBHandler) AddMetas(ctx *actor.ActionContext) error {
	handler.op.AddMetas(ctx)
	return nil
}
func (handler *DBHandler) sendAsyncUrlUpdate(ctx *actor.ActionContext) error {
	handler.op.sendAsyncUrlUpdate(ctx)

	return nil
}
func (handler *DBHandler) Initialization(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	if !handler.initDb {
		if msg.Name == scommon.MsgInitialization {
			handler.StateStore = msg.Data.(*mtypes.Initialization).Store

			handler.op.Init(handler.StateStore)
			handler.state = dbStateInit
			ctx.ExecCtx.LogDebug("change into dbStateInit,ready")
			handler.op.InitAsync(ctx)
		}
	} else {
		handler.state = dbStateInit
		ctx.ExecCtx.LogDebug("change into dbStateInit,ready")
		handler.op.InitAsync(ctx)
	}

	return nil
}

func (handler *DBHandler) importData(ctx *actor.ActionContext) error {
	data := ctx.Messages[0].Data.(*eushared.Euresults)
	_, transitions := GetTransitions(*data)
	handler.op.Import(transitions)
	return nil
}

func (handler *DBHandler) startCommit(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	combined := msg.Data.(*actor.CombinerElements)
	commitData := combined.Get(handler.commitMsg)
	conflictData := combined.Get(scommon.MsgConflictTransitions)

	var data []*eushared.EuResult
	if commitData.Data != nil {
		for _, item := range commitData.Data.([]interface{}) {
			data = append(data, item.(*eushared.EuResult))
		}
	}
	if commitData.Height == 0 {
		_, transitions := GetTransitions(data)
		handler.op.Import(transitions)
	}

	if handler.saveConfliction {
		conflictInfo := conflictData.Data.([]*statecell.StateCell)
		if len(conflictInfo) > 0 {
			handler.op.Import(conflictInfo)
		}
	}

	ctx.ExecCtx.LogDebug("Before PreCommit.")
	handler.op.PreCommit(ctx, data, commitData.Height)

	handler.AddAsyncTask(ctx)

	return nil
}

func (handler *DBHandler) generationComplete(ctx *actor.ActionContext) error {
	handler.state = dbStateDone
	ctx.ExecCtx.LogDebug("change into dbStateDone")

	handler.AddAsyncTask(ctx)

	return nil
}

func (handler *DBHandler) AddAsyncTask(ctx *actor.ActionContext) error {
	handler.taskCh <- &CommitTask{
		Msg: ctx.Messages[0],
		Ctx: ctx.ExecCtx.Fork(),
	}
	return nil
}

func (handler *DBHandler) finalize(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Before Commit.")
	handler.op.Commit(ctx, msg.Height)
	ctx.ExecCtx.LogDebug("After Commit.")
	handler.state = dbStateInit
	ctx.ExecCtx.LogDebug("change into dbStateInit")

	handler.AddAsyncTask(ctx)
	return nil
}

func (handler *DBHandler) GetFSMRules() map[int]actor.FSMRule {
	return map[int]actor.FSMRule{
		dbStateUninit: {Accept: []string{
			scommon.MsgInitialization,
		}},
		dbStateInit: {Accept: []string{
			scommon.MsgEuResults,
			actor.CombinedName(handler.commitMsg, scommon.MsgConflictTransitions),
			handler.generationCompletedMsg,
		}},
		dbStateDone: {Accept: []string{
			handler.finalizeMsg,
		}},
	}
}

func (handler *DBHandler) GetCurrentState() int {
	return handler.state
}
