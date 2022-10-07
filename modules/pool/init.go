package pool

import (
	"github.com/HPISTechnologies/component-lib/actor"
)

func init() {
	actor.Factory.Register("pool_aggr_selector", NewAggreSelector)
}
