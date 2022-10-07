package arbitrator

import (
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
)

func init() {
	actor.Factory.Register("access_record_pre_processor", NewEuResultPreProcessor)
	actor.Factory.Register("access_record_aggr_selector", NewEuResultsAggreSelector)
	actor.Factory.Register("arbitrator_rpc", NewRpcService)

	intf.Factory.Register("arbitrator_rpc", func(concurrency int, groupId string) interface{} {
		return NewRpcService(concurrency, groupId)
	})
}
