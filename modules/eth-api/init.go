package ethapi

import (
	"github.com/arcology-network/component-lib/actor"
)

func init() {
	actor.Factory.Register("eth_api", NewFilterManager)
}
