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

	eushared "github.com/arcology-network/eu/shared"
	stgcommon "github.com/arcology-network/storage-committer/common"
	"github.com/arcology-network/storage-committer/type/univalue"
	"github.com/arcology-network/streamer/actor"
)

type Feedback struct {
	actor.WorkerThread

	feeds []*univalue.Univalue
}

const (
	FEEDCOUNTS = 500
)

// return a Subscriber struct
func NewFeedback(concurrency int, groupid string) actor.IWorkerEx {
	fd := &Feedback{}
	fd.Set(concurrency, groupid)
	fd.feeds = make([]*univalue.Univalue, 0, FEEDCOUNTS)
	return fd
}

func (fd *Feedback) Inputs() ([]string, bool) {
	return []string{
		actor.MsgExecuted,
		actor.MsgBlockEnd,
	}, false
}

func (fd *Feedback) Outputs() map[string]int {
	return map[string]int{
		actor.MsgFeedBacks: 1,
	}
}

func (fd *Feedback) OnStart() {
}

func (fd *Feedback) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgExecuted:
			var data []*eushared.EuResult
			if v.Data != nil {
				for _, item := range v.Data.([]interface{}) {
					data = append(data, item.(*eushared.EuResult))
				}
			}
			fd.addUnivalues(data)
		case actor.MsgBlockEnd:
			fd.MsgBroker.Send(actor.MsgFeedBacks, fd.feeds)
			fd.feeds = make([]*univalue.Univalue, 0, FEEDCOUNTS)
		}
	}
	return nil
}
func (fd *Feedback) addUnivalues(data []*eushared.EuResult) error {
	for i := range data {
		for j := range data[i].Trans {
			if data[i].Trans[j].GetPath() != nil && strings.Contains(*data[i].Trans[j].GetPath(), stgcommon.FULL_FUNC_PATH) {
				fd.feeds = append(fd.feeds, data[i].Trans[j])
			}
		}
	}
	return nil
}
