package scheduler

import (
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
)

func init() {
	actor.Factory.Register("scheduler", NewScheduler)

	intf.Factory.Register("scheduler", func(concurrency int, groupId string) interface{} {
		return NewScheduler(concurrency, groupId)
	})
}
