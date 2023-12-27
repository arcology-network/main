package core

import (
	"github.com/arcology-network/component-lib/actor"
)

func init() {
	actor.Factory.Register("core.initializer", NewInitializer)
	// actor.Factory.Register("calc_tx_hash", NewCalculateTxHash)
	actor.Factory.Register("make_block", NewMakeBlock)
}
