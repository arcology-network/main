package scheduler

import (
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
)

func init() {
	actor.Factory.Register("scheduler", NewScheduler)

	intf.Factory.Register("scheduler", func(concurrency int, groupId string) interface{} {
		return NewScheduler(concurrency, groupId)
	})
}
