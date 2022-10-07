package tpp

import (
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
)

func init() {
	actor.Factory.Register("tpp_rpc", NewRpcReceiver)
	actor.Factory.Register("tx_receiver", NewTxReceiver)
	actor.Factory.Register("tx_unsigner", NewTxUnsigner)

	intf.Factory.Register("tpp_rpc", func(concurrency int, groupId string) interface{} {
		return NewRpcReceiver(concurrency, groupId)
	})
}
