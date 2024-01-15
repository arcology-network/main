package gateway

import (
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

func init() {
	actor.Factory.Register("local_tx_receiver", NewLocalReceiver)
	actor.Factory.Register("tx_dup_checker", NewTxRepeatedChecker)

	intf.Factory.Register("gateway_rpc", func(concurrency int, groupId string) interface{} {
		return NewLocalReceiver(concurrency, groupId)
	})
}
