package types

import (
	"math/big"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/mempool"
	ctypes "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/concurrenturl/commutative"
	"github.com/arcology-network/concurrenturl/interfaces"
	univaluepk "github.com/arcology-network/concurrenturl/univalue"
	"github.com/holiman/uint256"
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

	univalues := univaluepk.UnivaluesDecode(ars.Accesses, uniPool.Get, nil)
	for _, uv := range univalues {
		univalue := uv.(interfaces.Univalue)
		per.Txs = append(per.Txs, univalue.GetTx())
		per.Paths = append(per.Paths, *univalue.GetPath())
		per.Reads = append(per.Reads, univalue.Reads())
		per.Writes = append(per.Writes, univalue.Writes())
		per.Composite = append(per.Composite, true) //univalue.Composite())
		switch v := univalue.Value().(type) {
		case *commutative.U256:
			if (*uint256.Int)(v.Delta().(*codec.Uint256)).ToBig().Sign() >= 0 {
				continue
			}
			per.Transitions = append(per.Transitions, &BalanceTransition{
				Path:   *univalue.GetPath(),
				Tx:     univalue.GetTx(),
				Origin: (*uint256.Int)(v.Value().(*codec.Uint256)).ToBig(),
				Delta:  (*uint256.Int)(v.Delta().(*codec.Uint256)).ToBig(),
			})
		}
	}
	uniPool.Reclaim()
	return per
}
