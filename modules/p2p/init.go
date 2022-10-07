package p2p

import (
	"encoding/gob"

	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
)

func init() {
	actor.Factory.Register("p2p.gateway", NewP2pGateway)
	actor.Factory.Register("p2p.conn", NewP2pConn)
	actor.Factory.Register("p2p.client", NewP2pClient)

	intf.Factory.Register("p2p.conn", func(concurrency int, groupId string) interface{} {
		return NewP2pConn(concurrency, groupId)
	})

	gob.Register(&P2pMessage{})
}
