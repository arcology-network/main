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
	statestore "github.com/arcology-network/state-engine"

	statecell "github.com/arcology-network/common-lib/crdt/statecell"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/state-engine/storage/proxy"
)

type DBOperation interface {
	Init(stateStore *statestore.StateStore)
	InitAsync(ctx *actor.ActionContext)
	Import(transitions []*statecell.StateCell)
	PreCommit(ctx *actor.ActionContext, euResults []*eushared.EuResult, height uint64)
	PreCommitCompleted(ctx *actor.ActionContext)
	Commit(ctx *actor.ActionContext, height uint64)
	Outputs() map[string]int
	Config(params map[string]interface{})

	//for rpc async callback
	AddMetas(ctx *actor.ActionContext)
	sendAsyncUrlUpdate(ctx *actor.ActionContext)
}

type BasicDBOperation struct {
	StateStore *statestore.StateStore
	// MsgBroker  *actor.MessageWrapper

	Keys     []string
	Values   []interface{}
	AcctRoot [32]byte
}

func (op *BasicDBOperation) Init(stateStore *statestore.StateStore) {
	op.StateStore = stateStore
	op.Keys = []string{}
	op.Values = []interface{}{}
	op.AcctRoot = [32]byte{}
}

func (op *BasicDBOperation) Import(transitions []*statecell.StateCell) {
	op.StateStore.Import(transitions)
}

func (op *BasicDBOperation) PreCommit(euResults []*eushared.EuResult, height uint64) {
	op.StateStore.Finalize(GetTransitionIds(euResults))
	op.StateStore.SyncPrecommit()
	op.Keys = []string{}
	op.Values = []interface{}{}
}

func (op *BasicDBOperation) PreCommitCompleted() {

}
func (op *BasicDBOperation) InitAsync() {

}

func (op *BasicDBOperation) Commit(height uint64) {
	op.StateStore.SyncCommit(height)
}

func (op *BasicDBOperation) Outputs() map[string]int {
	return map[string]int{}
}

func (op *BasicDBOperation) Config(params map[string]interface{}) {}

const (
	dbStateUninit = iota
	dbStateInit
	dbStateDone
)

type DBHandler struct {
	StateStore             *statestore.StateStore
	state                  int
	importMsg              string
	commitMsg              string
	generationCompletedMsg string
	finalizeMsg            string
	op                     DBOperation

	initDb bool
}

func NewDBHandler(importMsg, commitMsg, generationCompletedMsg, finalizeMsg string, op DBOperation) *DBHandler {
	handler := &DBHandler{
		state:                  dbStateUninit,
		importMsg:              importMsg,
		commitMsg:              commitMsg,
		generationCompletedMsg: generationCompletedMsg,
		finalizeMsg:            finalizeMsg,
		op:                     op,
		initDb:                 false,
	}
	return handler
}

func (handler *DBHandler) Inputs() ([]string, bool) {
	msgs := []string{handler.importMsg, handler.commitMsg, handler.generationCompletedMsg, handler.finalizeMsg}
	if handler.state == dbStateUninit {
		msgs = append(msgs, scommon.MsgInitialization)
	}
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

			handler.StateStore = statestore.NewStateStore(proxy.NewLevelDBStoreProxy(dbpath, dbpath, math.MaxUint64, &hashdb.Config{CleanCacheSize: 1024 * 1024 * 100}))

			handler.op.Init(handler.StateStore)
			handler.initDb = true
		}
	}
	handler.op.Config(params)
}

func (handler *DBHandler) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgInitialization, handler.Initialization)
	reg.Register(handler.importMsg, handler.importData)
	reg.Register(handler.commitMsg, handler.startCommit)
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
	ctx.ExecCtx.LogDebug("After PreCommit.")
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
	var data []*eushared.EuResult
	if msg.Data != nil {
		for _, item := range msg.Data.([]interface{}) {
			data = append(data, item.(*eushared.EuResult))
		}
	}
	if msg.Height == 0 {
		_, transitions := GetTransitions(data)
		handler.op.Import(transitions)
	}
	ctx.ExecCtx.LogDebug("Before PreCommit.")
	handler.op.PreCommit(ctx, data, msg.Height)

	return nil
}

func (handler *DBHandler) generationComplete(ctx *actor.ActionContext) error {
	handler.op.PreCommitCompleted(ctx)
	handler.state = dbStateDone
	ctx.ExecCtx.LogDebug("change into dbStateDone")
	return nil
}

func (handler *DBHandler) finalize(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Before Commit.")
	handler.op.Commit(ctx, msg.Height)
	ctx.ExecCtx.LogDebug("After Commit.")
	handler.state = dbStateInit
	ctx.ExecCtx.LogDebug("change into dbStateInit")
	return nil
}

func (handler *DBHandler) GetFSMRules() map[int]actor.FSMRule {
	return map[int]actor.FSMRule{
		dbStateUninit: {Accept: []string{
			scommon.MsgInitialization,
		}},
		dbStateInit: {Accept: []string{
			scommon.MsgEuResults,
			handler.commitMsg,
			handler.generationCompletedMsg,
		}},
		dbStateDone: {Accept: []string{
			// actor.MsgEuResults,
			handler.finalizeMsg,
		}},
	}
}

func (handler *DBHandler) GetCurrentState() int {
	return handler.state
}
