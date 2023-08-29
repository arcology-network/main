package types

import (
	"github.com/arcology-network/common-lib/mempool"
	univaluepk "github.com/arcology-network/concurrenturl/univalue"
)

var UnivaluePool *mempool.Mempool
var RecordPool *mempool.Mempool

func init() {
	UnivaluePool = mempool.NewMempool("univalue", func() interface{} {
		return &univaluepk.Univalue{}
	})

	RecordPool = mempool.NewMempool("processed-result", func() interface{} {
		return &AccessRecord{}
	})

}
