package types

import (
	"github.com/arcology-network/common-lib/mempool"
	ctypes "github.com/arcology-network/common-lib/types"
	univaluepk "github.com/arcology-network/concurrenturl/univalue"
	"github.com/arcology-network/vm-adaptor/execution"
)

func Decode(ars *ctypes.TxAccessRecords, perPool, uniPool *mempool.Mempool) *execution.Result {
	result := perPool.Get().(*execution.Result)
	result.Transitions = univaluepk.UnivaluesDecode(ars.Accesses, uniPool.Get, nil)
	result.TxHash = [32]byte([]byte(ars.Hash))
	uniPool.Reclaim()
	return result
}
