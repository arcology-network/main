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

package pool

import (
	"github.com/arcology-network/main/components/storage"
	"github.com/arcology-network/streamer/actor"
	aggr "github.com/arcology-network/streamer/aggregator/v3"
	scommon "github.com/arcology-network/streamer/common"
)

func init() {
	actor.Factory.Register("pool_aggr_selector", NewAggrSelector)
	actor.Factory.Register("nonce_url", func() actor.Business {
		return storage.NewDBHandler(scommon.MsgNonceEuResults, scommon.MsgCommitNonceUrl, scommon.MsgGenerationReapingCompleted, scommon.MsgBlockEnd,
			storage.NewGeneralUrl(scommon.MsgNonceReady, scommon.MsgNonceDB, scommon.MsgNonceCompleted, scommon.MsgNoncePrecommit, scommon.MsgNonceCommit), false)
	})

	actor.Factory.Register("nonce_url_async", func() actor.Business {
		return storage.NewDBHandlerAsync(scommon.MsgNonceDB, scommon.MsgNoncePrecommit, scommon.MsgNonceCommit, scommon.MsgNonceCompleted)
	})

	actor.Factory.Register("stateless_euresult_aggr_selector4pool", func() actor.Business {
		return aggr.NewAggrSelector(
			"stateless_euresult_aggr_selector4pool",
			scommon.MsgNonceEuResults,
			scommon.MsgGenerationReapingList,
			scommon.MsgBlockEnd,
			&aggr.EuResultOperation{},
		)
	})

	// intf.Factory.Register("pool", func(concurrency int, groupId string) interface{} {
	// 	return NewAggrSelector(concurrency, groupId)
	// })
}
