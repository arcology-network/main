package types

import (
	"github.com/arcology-network/common-lib/mempool"
	ctypes "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/concurrenturl/interfaces"
	univaluepk "github.com/arcology-network/concurrenturl/univalue"
)

func Decode(ars *ctypes.TxAccessRecords, recordPool, uniPool *mempool.Mempool) *AccessRecord {
	record := recordPool.Get().(*AccessRecord)
	record.Accesses = univaluepk.UnivaluesDecode(ars.Accesses, uniPool.Get, nil)
	record.TxHash = [32]byte([]byte(ars.Hash))
	record.TxID = ars.ID
	uniPool.Reclaim()
	return record
}

type AccessRecord struct {
	GroupID  uint32
	TxID     uint32
	TxHash   [32]byte
	Accesses []interfaces.Univalue
}
