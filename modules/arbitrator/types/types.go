package types

import (
	"math/big"

	"github.com/HPISTechnologies/common-lib/mempool"
	ctypes "github.com/HPISTechnologies/common-lib/types"
	urltype "github.com/HPISTechnologies/concurrenturl/v2/type"
	"github.com/HPISTechnologies/concurrenturl/v2/type/commutative"
)

type BalanceTransition struct {
	Path   string
	Tx     uint32
	Origin *big.Int
	Delta  *big.Int
}

type ProcessedEuResult struct {
	Hash        string
	Txs         []uint32
	Paths       []string
	Reads       []uint32
	Writes      []uint32
	Composite   []bool
	Transitions []*BalanceTransition
}

func (per *ProcessedEuResult) Reset(hash string) {
	per.Hash = hash
	per.Txs = per.Txs[:0]
	per.Paths = per.Paths[:0]
	per.Reads = per.Reads[:0]
	per.Writes = per.Writes[:0]
	per.Composite = per.Composite[:0]
	per.Transitions = per.Transitions[:0]
}

func Process(ars *ctypes.TxAccessRecords, perPool, uniPool *mempool.Mempool) *ProcessedEuResult {
	per := perPool.Get().(*ProcessedEuResult)
	per.Reset(ars.Hash)
	univalues := urltype.Univalues{}
	univalues = univalues.DecodeV2(ars.Accesses, uniPool.Get, nil)
	for _, uv := range univalues {
		univalue := uv.(*urltype.Univalue)
		per.Txs = append(per.Txs, univalue.GetTx())
		per.Paths = append(per.Paths, *univalue.GetPath())
		per.Reads = append(per.Reads, univalue.Reads())
		per.Writes = append(per.Writes, univalue.Writes())
		per.Composite = append(per.Composite, univalue.Composite())
		switch v := univalue.Value().(type) {
		case *commutative.Balance:
			if v.GetDelta().(*big.Int).Sign() >= 0 {
				continue
			}
			per.Transitions = append(per.Transitions, &BalanceTransition{
				Path:   *univalue.GetPath(),
				Tx:     univalue.GetTx(),
				Origin: v.Value().(*big.Int),
				Delta:  v.GetDelta().(*big.Int),
			})
		}
	}
	uniPool.Reclaim()
	return per
}
