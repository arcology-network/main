package types

import (
	eushared "github.com/arcology-network/eu/shared"
	univaluepk "github.com/arcology-network/storage-committer/univalue"
)

// func Decode(ars *eushared.TxAccessRecords, recordPool *mempool.Mempool[AccessRecord], uniPool *mempool.Mempool[univaluepk.Univalue]) *AccessRecord {
// 	record := recordPool.Get()
// 	record.Accesses = univaluepk.Univalues{}.DecodeWithMempool(ars.Accesses, uniPool.Get, nil).(univaluepk.Univalues)
// 	record.TxHash = [32]byte([]byte(ars.Hash))
// 	record.TxID = ars.ID
// 	uniPool.Reclaim()
// 	return record
// }

func Decode(ars *eushared.TxAccessRecords) *AccessRecord {
	record := &AccessRecord{}
	record.Accesses = ars.Accesses
	record.TxHash = [32]byte([]byte(ars.Hash))
	record.TxID = ars.ID
	// uniPool.Reclaim()
	return record
}

type AccessRecord struct {
	GroupID  uint32
	TxID     uint32
	TxHash   [32]byte
	Accesses []*univaluepk.Univalue
}
