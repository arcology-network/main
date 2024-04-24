package scheduler

import (
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

func init() {
	actor.Factory.Register("scheduler", NewScheduler)
	actor.Factory.Register("apc_switcher", NewApcSwitcher)

	intf.Factory.Register("scheduler", func(concurrency int, groupId string) interface{} {
		return NewScheduler(concurrency, groupId)
	})
}
