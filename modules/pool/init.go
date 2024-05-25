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
		return storage.NewDBHandler(concurrency, groupId, actor.MsgNonceEuResults, actor.MsgCommitNonceUrl, actor.MsgGenerationReapingCompleted, actor.MsgBlockEnd, actor.MsgInitDBNonce,
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
