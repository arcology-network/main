package types

import (
	"github.com/HPISTechnologies/common-lib/mempool"
	urltype "github.com/HPISTechnologies/concurrenturl/v2/type"
)

var ProcessedEuResultPool *mempool.Mempool
var UnivaluePool *mempool.Mempool

func init() {
	ProcessedEuResultPool = mempool.NewMempool("processed-eu-result", func() interface{} {
		return &ProcessedEuResult{}
	})
	UnivaluePool = mempool.NewMempool("univalue", func() interface{} {
		return &urltype.Univalue{}
	})
}
