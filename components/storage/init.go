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
	intf "github.com/arcology-network/streamer/interface"
)

func init() {
	actor.Factory.Register("gc", NewGc)
	actor.Factory.Register("general_url", func(concurrency int, groupId string) actor.IWorkerEx {
		return NewDBHandler(concurrency, groupId, actor.MsgEuResults, actor.MsgExecuted, actor.MsgGenerationReapingCompleted, actor.MsgBlockEnd,
			NewGeneralUrl(actor.MsgApcHandle, actor.MsgGeneralDB, actor.MsgGeneralCompleted, actor.MsgGeneralPrecommit, actor.MsgGeneralCommit))
	})
	actor.Factory.Register("general_url_async", func(concurrency int, groupId string) actor.IWorkerEx {
		return NewDBHandlerAsync(concurrency, groupId, actor.MsgGeneralDB, actor.MsgGeneralPrecommit, actor.MsgGeneralCommit, actor.MsgGeneralCompleted)
	})

	intf.Factory.Register("global_lock", func(int, string) interface{} {
		return NewModulesGuard()
	})
}
