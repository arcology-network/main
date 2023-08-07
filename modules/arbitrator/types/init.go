package types

import (
	"github.com/arcology-network/common-lib/mempool"
	univaluepk "github.com/arcology-network/concurrenturl/univalue"
	"github.com/arcology-network/vm-adaptor/execution"
)

// var ProcessedEuResultPool *mempool.Mempool
var UnivaluePool *mempool.Mempool
var ResultPool *mempool.Mempool

func init() {
	UnivaluePool = mempool.NewMempool("univalue", func() interface{} {
		return &univaluepk.Univalue{}
	})

	ResultPool = mempool.NewMempool("processed-result", func() interface{} {
		return &execution.Result{}
	})
}
