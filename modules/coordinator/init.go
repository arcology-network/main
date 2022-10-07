package coordinator

import (
	"github.com/HPISTechnologies/component-lib/actor"
)

func init() {
	actor.Factory.Register("coordinator.decision_maker", NewDecisionMaker)
}
