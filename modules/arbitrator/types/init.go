package types

import (
	"github.com/arcology-network/common-lib/mempool"
	univaluepk "github.com/arcology-network/concurrenturl/univalue"
)

var ProcessedEuResultPool *mempool.Mempool
var UnivaluePool *mempool.Mempool

func init() {
	ProcessedEuResultPool = mempool.NewMempool("processed-eu-result", func() interface{} {
		return &ProcessedEuResult{}
	})
	UnivaluePool = mempool.NewMempool("univalue", func() interface{} {
		return &univaluepk.Univalue{}
	})
}
