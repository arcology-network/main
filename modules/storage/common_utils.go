package storage

import (
	ethcmn "github.com/HPISTechnologies/3rd-party/eth/common"
	cmncmn "github.com/HPISTechnologies/common-lib/common"
	"github.com/HPISTechnologies/common-lib/mempool"
	cmnmkl "github.com/HPISTechnologies/common-lib/merkle"
	"github.com/HPISTechnologies/common-lib/mhasher"
	urltyp "github.com/HPISTechnologies/concurrenturl/v2/type"
)

func calcRootHash(merkle *urltyp.AccountMerkle, lastRoot ethcmn.Hash, paths []string, encodedValues [][]byte) ethcmn.Hash {
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
	cmncmn.ParallelWorker(len(sortedKeys), 6, worker)

	all := cmnmkl.NewMerkle(len(rootDatas), cmnmkl.Sha256)
	nodePool := mempool.NewMempool("nodes", func() interface{} {
		return cmnmkl.NewNode()
	})
	all.Init(rootDatas, nodePool)
	return ethcmn.BytesToHash(all.GetRoot())
}
