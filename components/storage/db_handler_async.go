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
	"github.com/arcology-network/streamer/log"

	statestore "github.com/arcology-network/storage-committer"
)

const (
	dbStateWaitInit = iota
	dbStateCommit
)

type DBHandlerAsync struct {
	actor.WorkerThread

	StateStore             *statestore.StateStore
	state                  int
	dbhandle               string
	precommitMsg           string
	generationCompletedMsg string
	commitMsg              string

	generateAcctRoot bool

	taskCh chan *actor.Message
}

func NewDBHandlerAsync(concurrency int, groupId string, dbhandle, precommitMsg, commitMsg, generationCompletedMsg string) *DBHandlerAsync {
	handler := &DBHandlerAsync{
		dbhandle:               dbhandle,
		state:                  dbStateWaitInit,
		precommitMsg:           precommitMsg,
		commitMsg:              commitMsg,
		generationCompletedMsg: generationCompletedMsg,
		taskCh:                 make(chan *actor.Message, 20),
	}
	handler.Set(concurrency, groupId)
	return handler
}

func (handler *DBHandlerAsync) Inputs() ([]string, bool) {
	msgs := []string{handler.dbhandle, handler.precommitMsg, handler.commitMsg, handler.generationCompletedMsg}

	return msgs, false
}

func (handler *DBHandlerAsync) Outputs() map[string]int {
	outputs := make(map[string]int)
	if handler.generateAcctRoot {
		outputs[actor.MsgAcctHash] = 1
	}
	return outputs
}

func (handler *DBHandlerAsync) Config(params map[string]interface{}) {
	if v, ok := params["generate_acct_root"]; !ok {
		panic("parameter not found: generate_acct_root")
	} else {
		handler.generateAcctRoot = v.(bool)
	}
}

func (handler *DBHandlerAsync) OnStart() {
	go func() {
		for {
			msg := <-handler.taskCh

			if msg.Name == handler.precommitMsg {
				handler.AddLog(log.LogLevel_Info, "Before Precommit Async.")
				handler.StateStore.AsyncPrecommit()
				handler.AddLog(log.LogLevel_Info, "After Precommit Async.")
			} else if msg.Name == handler.generationCompletedMsg {
				if handler.generateAcctRoot {
					handler.MsgBroker.Send(actor.MsgAcctHash, handler.StateStore.Backend().EthStore().LatestWorldTrieRoot(), msg.Height)
				}
				handler.AddLog(log.LogLevel_Debug, ">>>>>change into dbStateCommit >>>>>>>>")
			} else if msg.Name == handler.commitMsg {
				handler.AddLog(log.LogLevel_Info, "Before Commit Async.")
				handler.StateStore.AsyncCommit(msg.Height)
				handler.AddLog(log.LogLevel_Info, "After Commit Async.")
				handler.AddLog(log.LogLevel_Debug, ">>>>>change into dbStatePrecommit >>>>>>>>")
			}

		}
	}()
}

func (handler *DBHandlerAsync) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	if handler.state == dbStateWaitInit {
		if msg.Name == handler.dbhandle {
			handler.StateStore = msg.Data.(*statestore.StateStore)
			handler.state = dbStateCommit
			handler.AddLog(log.LogLevel_Debug, ">>>>>change into dbStatePrecommit>>>>>>>>")
		}
	} else {
		if msg.Name == handler.precommitMsg ||
			msg.Name == handler.generationCompletedMsg ||
			msg.Name == handler.commitMsg {
			handler.taskCh <- msg
		}
	}
	return nil
}

func (handler *DBHandlerAsync) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		dbStateWaitInit: {handler.dbhandle},
		dbStateCommit:   {handler.precommitMsg, handler.generationCompletedMsg, handler.commitMsg},
	}
}

func (handler *DBHandlerAsync) GetCurrentState() int {
	return handler.state
}
