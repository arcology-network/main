package ethapi

import (
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
)

func init() {
	actor.Factory.Register("eth_api", NewFilterManager)
	actor.Factory.Register("state_query", NewStateQuery)
	intf.Factory.Register("state_query", func(concurrency int, groupId string) interface{} {
		return NewStateQuery(concurrency, groupId)
	})
}
