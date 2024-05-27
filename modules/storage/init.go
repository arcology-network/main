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
	"encoding/gob"

	"github.com/arcology-network/common-lib/storage/transactional"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

func init() {
	gob.Register(&SchdState{})
	gob.Register([]SchdState{})

	actor.Factory.Register("storage", NewStorage)
	// actor.Factory.Register("storage_debug", NewStorageDebug)
	actor.Factory.Register("storage.initializer", NewInitializer)
	actor.Factory.Register("storage.metrics", NewMetrics)
	actor.Factory.Register("storage.statesyncstore", NewStateSyncStore)
	actor.Factory.Register("storage.schdstore", NewSchdStore)
	actor.Factory.Register("storage.cacheblockstore", NewCacheBlockStore)

	intf.Factory.Register("storage.tmblockstore", func(int, string) interface{} {
		return NewTmBlockStore()
	})
	intf.Factory.Register("storage.tmstatestore", func(int, string) interface{} {
		return NewTmStateStore()
	})
	intf.Factory.Register("storage.urlstore", func(int, string) interface{} {
		return NewUrlStore()
	})
	// intf.Factory.Register("storage.txstore", func(int, string) interface{} {
	// 	return NewTxStore()
	// })
	intf.Factory.Register("storage.receiptstore", func(int, string) interface{} {
		return NewReceiptStore()
	})
	intf.Factory.Register("storage.blockstore", func(int, string) interface{} {
		return NewBlockStore()
	})
	intf.Factory.Register("storage.statestore", func(int, string) interface{} {
		return NewStateStore()
	})
	intf.Factory.Register("storage.debugstore", func(int, string) interface{} {
		return NewDebugStore()
	})
	intf.Factory.Register("storage.indexerstore", func(int, string) interface{} {
		return NewIndexerStore()
	})
	intf.Factory.Register("storage.statesyncstore", func(concurrency int, groupId string) interface{} {
		return NewStateSyncStore(concurrency, groupId)
	})
	intf.Factory.Register("storage", func(concurrency int, groupId string) interface{} {
		return NewStorage(concurrency, groupId)
	})
	intf.Factory.Register("storage.transactionalstore", func(int, string) interface{} {
		return transactional.NewTransactionalStore()
	})
	intf.Factory.Register("storage.schdstore", func(concurrency int, groupId string) interface{} {
		return NewSchdStore(concurrency, groupId)
	})

}
