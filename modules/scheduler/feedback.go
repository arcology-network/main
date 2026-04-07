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
	"strings"

	statecell "github.com/arcology-network/common-lib/crdt/statecell"
	eushared "github.com/arcology-network/eu/shared"
	stgcommon "github.com/arcology-network/state-engine/common"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
)

type Feedback struct {
	feeds []*statecell.StateCell
}

const (
	FEEDCOUNTS = 500
)

// return a Subscriber struct
func NewFeedback() actor.Business {
	fd := &Feedback{}
	fd.feeds = make([]*statecell.StateCell, 0, FEEDCOUNTS)
	return fd
}

func (fd *Feedback) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgExecuted,
		scommon.MsgBlockEnd,
	}, false
}

func (fd *Feedback) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgFeedBacks: 1,
	}
}

func (fd *Feedback) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgExecuted, fd.receivedData)
	reg.Register(scommon.MsgBlockEnd, fd.receiveBlockEnd)
}

func (fd *Feedback) receivedData(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	var data []*eushared.EuResult
	if msg.Data != nil {
		for _, item := range msg.Data.([]interface{}) {
			data = append(data, item.(*eushared.EuResult))
		}
	}
	fd.addUnivalues(data)
	return nil
}

func (fd *Feedback) receiveBlockEnd(ctx *actor.ActionContext) error {
	ctx.ExecCtx.Send(scommon.MsgFeedBacks, fd.feeds)
	fd.feeds = make([]*statecell.StateCell, 0, FEEDCOUNTS)
	return nil
}

func (fd *Feedback) addUnivalues(data []*eushared.EuResult) error {
	for i := range data {
		for j := range data[i].Trans {
			if data[i].Trans[j].GetPath() != nil && strings.Contains(*data[i].Trans[j].GetPath(), stgcommon.PATH_FUNC_PROFILE) {
				fd.feeds = append(fd.feeds, data[i].Trans[j])
			}
		}
	}
	return nil
}
