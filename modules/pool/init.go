package pool

import (
	"github.com/arcology-network/component-lib/actor"
	aggr "github.com/arcology-network/component-lib/aggregator/v3"
	intf "github.com/arcology-network/component-lib/interface"
	"github.com/arcology-network/component-lib/storage"
)

func init() {
	actor.Factory.Register("pool_aggr_selector", NewAggrSelector)
	actor.Factory.Register("nonce_url", func(concurrency int, groupId string) actor.IWorkerEx {
		return storage.NewDBHandler(concurrency, groupId, actor.MsgCommitNonceUrl, actor.MsgBlockEnd, storage.NewGeneralUrl(actor.MsgNonceReady))
	})
	actor.Factory.Register("stateful_euresult_aggr_selector4pool", func(concurrency int, groupId string) actor.IWorkerEx {
		return aggr.NewStatefulAggrSelector(
			concurrency,
			groupId,
			actor.MsgEuResults,
			actor.MsgInclusive,
			actor.MsgBlockEnd,
			&aggr.EuResultOperation{},
		)
	})

	intf.Factory.Register("pool", func(concurrency int, groupId string) interface{} {
		return NewAggrSelector(concurrency, groupId)
	})
}
