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
	"github.com/arcology-network/common-lib/crdt/statecell"
	"github.com/arcology-network/streamer/actor"

	scommon "github.com/arcology-network/streamer/common"
)

type ConflictAssistant struct {
	commitMsg string
}

func NewConflictAssistant(commitMsg string) *ConflictAssistant {
	handler := &ConflictAssistant{
		commitMsg: commitMsg,
	}
	return handler
}

func (handler *ConflictAssistant) Inputs() ([]string, bool) {
	msgs := []string{handler.commitMsg}

	return msgs, false
}

func (handler *ConflictAssistant) Outputs() map[string]int {
	outputs := make(map[string]int)
	outputs[scommon.MsgConflictTransitions] = 1
	return outputs
}

func (handler *ConflictAssistant) Config(params map[string]interface{}) {
}

func (handler *ConflictAssistant) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(handler.commitMsg, handler.assistant)
}

func (handler *ConflictAssistant) assistant(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	if len(msg.Data.([]interface{})) == 0 {
		ctx.ExecCtx.Send(scommon.MsgConflictTransitions, []*statecell.StateCell{})
	}
	return nil
}
