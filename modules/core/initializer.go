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

package core

import (
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

type Initializer struct {
	actor.WorkerThread

	inited bool
}

func NewInitializer(concurrency int, groupId string) actor.IWorkerEx {
	w := &Initializer{}
	w.Set(concurrency, groupId)
	return w
}

func (i *Initializer) Inputs() ([]string, bool) {
	return []string{actor.MsgBlockCompleted}, false
}

func (i *Initializer) Outputs() map[string]int {
	return map[string]int{
		actor.MsgLocalParentInfo: 1,
		actor.MsgParentInfo:      1,
	}
}

func (i *Initializer) OnStart() {}

func (i *Initializer) OnMessageArrived(msgs []*actor.Message) error {
	if !i.inited {
		var parentInfo mtypes.ParentInfo
		iin := 12
		if err := intf.Router.Call("statestore", "GetParentInfo", &iin, &parentInfo); err != nil {
			panic(err)
		}
		//fmt.Printf("[core.Initializer.OnMessageArrived] init parent info: %v\n", parentInfo)
		i.MsgBroker.Send(actor.MsgLocalParentInfo, &parentInfo)
		i.MsgBroker.Send(actor.MsgParentInfo, &parentInfo)
		i.inited = true
	}
	return nil
}
