package coordinator

import (
	"github.com/arcology-network/streamer/actor"
)

func init() {
	actor.Factory.Register("coordinator.decision_maker", NewDecisionMaker)
}
