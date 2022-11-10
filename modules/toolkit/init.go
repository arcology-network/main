package toolkit

import (
	"github.com/arcology-network/component-lib/actor"
)

func init() {
	actor.Factory.Register("read_kafka", Newkafka)
}
