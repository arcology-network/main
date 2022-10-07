package receipthashing

import (
	"github.com/HPISTechnologies/component-lib/actor"
)

func init() {
	actor.Factory.Register("receipt-hashing.calculator", NewCalculateRoothash)
}
