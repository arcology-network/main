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
	scommon "github.com/arcology-network/streamer/common"
)

func init() {
	actor.Factory.Register("gc", NewGc)
	actor.Factory.Register("general_url", func() actor.Business {
		return NewDBHandler(scommon.MsgEuResults, scommon.MsgExecuted, scommon.MsgGenerationReapingCompleted, scommon.MsgBlockEnd,
			NewGeneralUrl(scommon.MsgApcHandle, scommon.MsgGeneralDB, scommon.MsgGeneralCompleted, scommon.MsgGeneralPrecommit, scommon.MsgGeneralCommit))
	})
	actor.Factory.Register("general_url_async", func() actor.Business {
		return NewDBHandlerAsync(scommon.MsgGeneralDB, scommon.MsgGeneralPrecommit, scommon.MsgGeneralCommit, scommon.MsgGeneralCompleted)
	})

	// intf.Factory.Register("global_lock", func(int, string) interface{} {
	// 	return NewModulesGuard()
	// })
}
