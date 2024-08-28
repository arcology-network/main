/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package types

import (
	eushared "github.com/arcology-network/eu/shared"
	univaluepk "github.com/arcology-network/storage-committer/type/univalue"
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
