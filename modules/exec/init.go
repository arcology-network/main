package exec

import (
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

func init() {
	actor.Factory.Register("exec_rpc", NewRpcService)
	actor.Factory.Register("executor", NewExecutor)

	intf.Factory.Register("exec_rpc", func(concurrency int, groupId string) interface{} {
		return NewRpcService(concurrency, groupId)
	})
}
