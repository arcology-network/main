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
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

type ApcSwitcher struct {
	actor.WorkerThread
}

func NewApcSwitcher(concurrency int, groupId string) actor.IWorkerEx {
	apcSwitcher := &ApcSwitcher{}
	apcSwitcher.Set(concurrency, groupId)
	return apcSwitcher
}

func (r *ApcSwitcher) Inputs() ([]string, bool) {
	return []string{
		actor.MsgApcHandle, // Init DB on every generation.
	}, false
}

func (r *ApcSwitcher) Outputs() map[string]int {
	return map[string]int{}
}

func (r *ApcSwitcher) Config(params map[string]interface{}) {

}

func (r *ApcSwitcher) OnStart() {
}

func (r *ApcSwitcher) OnMessageArrived(msgs []*actor.Message) error {
	switch msgs[0].Name {
	case actor.MsgApcHandle:
		var na int
		intf.Router.Call("scheduler", "NotifyApchandler", msgs[0].Height, &na)
	}
	return nil
}
