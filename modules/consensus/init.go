package consensus

import (
	"github.com/HPISTechnologies/common-lib/mhasher"
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
	"github.com/HPISTechnologies/consensus-engine/types"
)

func init() {
	types.QuickHash = BinaryMerkleFromRaw256

	actor.Factory.Register("consensus", NewConsensus)
	intf.Factory.Register("consensus", func(concurrency int, groupId string) interface{} {
		return NewConsensus(concurrency, groupId)
	})
}

func BinaryMerkleFromRaw256(txs types.Txs) ([]byte, error) {
	datas := make([][]byte, len(txs))
	sizes := make([]int, len(txs))
	for i := range txs {
		datas[i] = txs[i]
		sizes[i] = len(txs[i])
	}
	return mhasher.Roothash(datas, mhasher.HashType_256)
}
