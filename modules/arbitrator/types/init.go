package types

import (
	"github.com/arcology-network/common-lib/mempool"
	univaluepk "github.com/arcology-network/storage-committer/univalue"
)

var UnivaluePool *mempool.Mempool[univaluepk.Univalue]
var RecordPool *mempool.Mempool[AccessRecord]

func init() {
	UnivaluePool = mempool.NewMempool[univaluepk.Univalue]("univalue", func() *univaluepk.Univalue {
		return &univaluepk.Univalue{}
	})

	RecordPool = mempool.NewMempool("processed-result", func() *AccessRecord {
		return &AccessRecord{}
	})

}
