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
	"github.com/arcology-network/streamer/actor"

	statestore "github.com/arcology-network/storage-committer"
	scommon "github.com/arcology-network/streamer/common"
)

const (
	dbStateWaitInit = iota
	dbStateCommit
)

type DBTask struct {
	Msg     *scommon.Message
	ExecCtx *actor.ExecutionContext
}

type DBHandlerAsync struct {
	StateStore             *statestore.StateStore
	state                  int
	dbhandle               string
	precommitMsg           string
	generationCompletedMsg string
	commitMsg              string

	generateAcctRoot bool

	taskCh chan *DBTask
}

func NewDBHandlerAsync(dbhandle, precommitMsg, commitMsg, generationCompletedMsg string) *DBHandlerAsync {
	handler := &DBHandlerAsync{
		dbhandle:               dbhandle,
		state:                  dbStateWaitInit,
		precommitMsg:           precommitMsg,
		commitMsg:              commitMsg,
		generationCompletedMsg: generationCompletedMsg,
		taskCh:                 make(chan *DBTask, 20),
	}
	return handler
}

func (handler *DBHandlerAsync) Inputs() ([]string, bool) {
	msgs := []string{handler.dbhandle, handler.precommitMsg, handler.commitMsg, handler.generationCompletedMsg}

	return msgs, false
}

func (handler *DBHandlerAsync) Outputs() map[string]int {
	outputs := make(map[string]int)
	if handler.generateAcctRoot {
		outputs[scommon.MsgAcctHash] = 1
	}
	return outputs
}

func (handler *DBHandlerAsync) Config(params map[string]interface{}) {

	if v, ok := params["generate_acct_root"]; !ok {
		panic("parameter not found: generate_acct_root")
	} else {
		handler.generateAcctRoot = v.(bool)
	}

	go func() {
		for {
			task := <-handler.taskCh

			switch task.Msg.Name {
			case handler.precommitMsg:
				task.ExecCtx.LogDebug("Before Precommit Async.")
				handler.StateStore.AsyncPrecommit()
				task.ExecCtx.LogDebug("After Precommit Async.")
			case handler.generationCompletedMsg:
				if handler.generateAcctRoot {
					task.ExecCtx.Send(scommon.MsgAcctHash, handler.StateStore.Backend().EthStore().LatestWorldTrieRoot(), task.Msg.Height)
				}
				task.ExecCtx.LogDebug("change into dbStateCommit")
			case handler.commitMsg:
				task.ExecCtx.LogDebug("Before Commit Async.")
				handler.StateStore.AsyncCommit(task.Msg.Height)
				task.ExecCtx.LogDebug("After Commit Async.")
				task.ExecCtx.LogDebug("change into dbStatePrecommit")
			}

		}
	}()
}

func (handler *DBHandlerAsync) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(handler.dbhandle, handler.Initialization)
	reg.Register(handler.precommitMsg, handler.dbAsync)
	reg.Register(handler.generationCompletedMsg, handler.dbAsync)
	reg.Register(handler.commitMsg, handler.dbAsync)
}

func (handler *DBHandlerAsync) Initialization(ctx *actor.ActionContext) error {
	handler.StateStore = ctx.Messages[0].Data.(*statestore.StateStore)
	handler.state = dbStateCommit
	ctx.ExecCtx.LogDebug("change into dbStatePrecommit,ready")
	return nil
}

func (handler *DBHandlerAsync) dbAsync(ctx *actor.ActionContext) error {
	handler.taskCh <- &DBTask{
		Msg:     ctx.Messages[0],
		ExecCtx: ctx.ExecCtx.Fork(),
	}
	return nil
}

func (handler *DBHandlerAsync) GetFSMRules() map[int]actor.FSMRule {
	return map[int]actor.FSMRule{
		dbStateWaitInit: {Accept: []string{
			handler.dbhandle,
		}},
		dbStateCommit: {Accept: []string{
			handler.precommitMsg,
			handler.generationCompletedMsg,
			handler.commitMsg,
		}},
	}
}

func (handler *DBHandlerAsync) GetCurrentState() int {
	return handler.state
}
