package arbitrator

import (
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

func init() {
	actor.Factory.Register("access_record_pre_processor", NewEuResultPreProcessor)
	actor.Factory.Register("access_record_aggr_selector", NewEuResultsAggreSelector)
	actor.Factory.Register("arbitrator_rpc", NewRpcService)

	intf.Factory.Register("arbitrator_rpc", func(concurrency int, groupId string) interface{} {
		return NewRpcService(concurrency, groupId)
	})
}
