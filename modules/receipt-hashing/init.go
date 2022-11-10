package receipthashing

import (
	"github.com/arcology-network/component-lib/actor"
)

func init() {
	actor.Factory.Register("receipt-hashing.calculator", NewCalculateRoothash)
}
