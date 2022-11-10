package coordinator

import (
	"github.com/arcology-network/component-lib/actor"
)

func init() {
	actor.Factory.Register("coordinator.decision_maker", NewDecisionMaker)
}
