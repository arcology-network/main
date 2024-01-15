package toolkit

import (
	"github.com/arcology-network/streamer/actor"
)

func init() {
	actor.Factory.Register("read_kafka", Newkafka)
}
