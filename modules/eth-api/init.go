package ethapi

import (
	"github.com/HPISTechnologies/component-lib/actor"
)

func init() {
	actor.Factory.Register("eth_api", NewFilterManager)
}
