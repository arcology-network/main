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
	"sync"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

var (
	cbsSingleton actor.IWorkerEx
	initCbs      sync.Once
)

type CacheBlockStore struct {
	actor.WorkerThread
}

func NewCacheBlockStore(concurrency int, groupId string) actor.IWorkerEx {
	initCbs.Do(func() {
		cbsSingleton = &CacheBlockStore{}
		cbsSingleton.(*CacheBlockStore).Set(concurrency, groupId)
	})
	return cbsSingleton
}

func (cbs *CacheBlockStore) Inputs() ([]string, bool) {
	return []string{actor.MsgPendingBlock}, false
}

func (cbs *CacheBlockStore) Outputs() map[string]int {
	return map[string]int{}
}

func (cbs *CacheBlockStore) OnStart() {}

func (cbs *CacheBlockStore) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch msg.Name {
	case actor.MsgPendingBlock:
		block := msg.Data.(*mtypes.MonacoBlock)
		var na int
		intf.Router.Call("blockstore", "SavePendingBlock", block, &na)
	}
	return nil
}
