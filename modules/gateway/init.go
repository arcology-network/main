package gateway

import (
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
)

func init() {
	actor.Factory.Register("local_tx_receiver", NewLocalReceiver)
	actor.Factory.Register("tx_dup_checker", NewTxRepeatedChecker)
	actor.Factory.Register("tpp_rpc_client", NewRpcClient)

	intf.Factory.Register("gateway_rpc", func(concurrency int, groupId string) interface{} {
		return NewLocalReceiver(concurrency, groupId)
	})
}
