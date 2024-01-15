package consensus

import (
	"crypto/sha256"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/consensus-engine/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

func init() {
	types.QuickHash = CalculateHash

	actor.Factory.Register("consensus", NewConsensus)
	intf.Factory.Register("consensus", func(concurrency int, groupId string) interface{} {
		return NewConsensus(concurrency, groupId)
	})
}

func CalculateHash(txs types.Txs) ([]byte, error) {
	data := make([][]byte, len(txs))
	for i := range txs {
		data[i] = txs[i]
	}
	hash := sha256.Sum256(codec.Byteset(data).Flatten())
	return hash[:], nil
}
