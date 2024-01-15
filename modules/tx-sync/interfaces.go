package txsync

import (
	"github.com/arcology-network/streamer/actor"
)

type P2pClient interface {
	ID() string
	Broadcast(msg *actor.Message)
	Request(peer string, msg *actor.Message)
	Response(peer string, msg *actor.Message)
	OnConnClosed(cb func(id string))
}
