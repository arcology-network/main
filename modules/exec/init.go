package exec

import (
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
)

func init() {
	actor.Factory.Register("exec_rpc", NewRpcService)
	actor.Factory.Register("snapshot_maker", NewSnapshotMaker)
	actor.Factory.Register("executor", NewExecutor)

	intf.Factory.Register("exec_rpc", func(concurrency int, groupId string) interface{} {
		return NewRpcService(concurrency, groupId)
	})
}
