package statesync

import (
	"github.com/arcology-network/streamer/actor"
)

func init() {
	actor.Factory.Register("statesync.client", NewSyncClient)
	actor.Factory.Register("statesync.server", NewSyncServer)
}
