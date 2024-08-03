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
	intf "github.com/arcology-network/streamer/interface"
)

func init() {
	actor.Factory.Register("pool_aggr_selector", NewAggrSelector)
	actor.Factory.Register("nonce_url", func(concurrency int, groupId string) actor.IWorkerEx {
		return storage.NewDBHandler(concurrency, groupId, actor.MsgNonceEuResults, actor.MsgCommitNonceUrl, actor.MsgGenerationReapingCompleted, actor.MsgBlockEnd,
			storage.NewGeneralUrl(actor.MsgNonceReady, actor.MsgNonceDB, actor.MsgNonceCompleted, actor.MsgNoncePrecommit, actor.MsgNonceCommit))
	})

	actor.Factory.Register("nonce_url_async", func(concurrency int, groupId string) actor.IWorkerEx {
		return storage.NewDBHandlerAsync(concurrency, groupId, actor.MsgNonceDB, actor.MsgNoncePrecommit, actor.MsgNonceCommit, actor.MsgNonceCompleted)
	})

	actor.Factory.Register("stateless_euresult_aggr_selector4pool", func(concurrency int, groupId string) actor.IWorkerEx {
		return aggr.NewAggrSelector(
			concurrency,
			groupId,
			actor.MsgNonceEuResults,
			actor.MsgGenerationReapingList,
			actor.MsgBlockEnd,
			&aggr.EuResultOperation{},
		)
	})

	intf.Factory.Register("pool", func(concurrency int, groupId string) interface{} {
		return NewAggrSelector(concurrency, groupId)
	})
}
