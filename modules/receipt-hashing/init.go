package receipthashing

import (
	"github.com/arcology-network/streamer/actor"
)

func init() {
	actor.Factory.Register("receipt-hashing.calculator", NewCalculateRoothash)
}
