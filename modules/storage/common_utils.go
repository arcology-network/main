package storage

import (
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/mempool"
	cmnmkl "github.com/arcology-network/common-lib/merkle"
	"github.com/arcology-network/common-lib/mhasher"
	"github.com/arcology-network/concurrenturl/indexer"
	evmCommon "github.com/arcology-network/evm/common"
)

func calcRootHash(merkle *indexer.AccountMerkle, lastRoot evmCommon.Hash, paths []string, encodedValues [][]byte) evmCommon.Hash {
	merkle.Build(paths, encodedValues)

	merkles := merkle.GetMerkles()
	if len(*merkles) == 0 {
		return lastRoot
	}

	keys := make([]string, 0, len(*merkles))
	for p := range *merkles {
		keys = append(keys, p)
	}
	sortedKeys, err := mhasher.SortStrings(keys)
	if err != nil {
		panic(err)
	}

	rootDatas := make([][]byte, len(sortedKeys))
	worker := func(start, end, index int, args ...interface{}) {
		for i := start; i < end; i++ {
			if merkle, ok := (*merkles)[sortedKeys[i]]; ok {
				rootDatas[i] = merkle.GetRoot()
			}
		}
	}
	common.ParallelWorker(len(sortedKeys), 6, worker)

	all := cmnmkl.NewMerkle(len(rootDatas), cmnmkl.Concatenator{}, cmnmkl.Sha256{})
	nodePool := mempool.NewMempool("nodes", func() interface{} {
		return cmnmkl.NewNode()
	})
	all.Init(rootDatas, nodePool)
	return evmCommon.BytesToHash(all.GetRoot())
}
