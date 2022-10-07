package statesync

import (
	"github.com/HPISTechnologies/component-lib/actor"
)

func init() {
	actor.Factory.Register("statesync.client", NewSyncClient)
	actor.Factory.Register("statesync.server", NewSyncServer)
}
