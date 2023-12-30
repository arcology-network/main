package txsync

import (
	"github.com/arcology-network/component-lib/actor"
)

func init() {
	actor.Factory.Register("txsync.reap_timeout_watcher", NewReapTimeoutWatcher)
	actor.Factory.Register("txsync.server", NewSyncServer)
}
