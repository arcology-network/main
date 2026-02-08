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
)

func init() {
	actor.Factory.Register("storage", NewStorage)
	actor.Factory.Register("storage.initializer", NewInitializer)
	actor.Factory.Register("storage.metrics", NewMetrics)
	// actor.Factory.Register("storage.statesyncstore", NewStateSyncStore)
	actor.Factory.Register("storage.schdstore", NewSchdStore)

	actor.Factory.Register("storage.tmblockstore", NewTmBlockStore)
	actor.Factory.Register("storage.tmstatestore", NewTmStateStore)
	actor.Factory.Register("storage.urlstore", NewUrlStore)
	actor.Factory.Register("storage.receiptstore", NewReceiptStore)
	actor.Factory.Register("storage.blockstore", NewBlockStore)
	actor.Factory.Register("storage.statestore", NewStateStore)
	actor.Factory.Register("storage.indexerstore", NewIndexerStore)
	actor.Factory.Register("storage.transactionalstore", NewTransactionalStore)

}
