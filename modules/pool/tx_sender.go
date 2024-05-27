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

package pool

import (
	"fmt"

	cmntyp "github.com/arcology-network/common-lib/types"
)

type TxSender struct {
	Nonce        uint64
	TxByNonce    map[uint64]*cmntyp.StandardTransaction
	ObsoleteTime uint64
	LatestHeight uint64
}

func NewTxSender(nonce uint64, obsoleteTime uint64) *TxSender {
	return &TxSender{
		Nonce:        nonce,
		TxByNonce:    make(map[uint64]*cmntyp.StandardTransaction),
		ObsoleteTime: obsoleteTime,
	}
}

func (s *TxSender) Add(txs []*cmntyp.StandardTransaction, stat *TxSourceStatistics, height uint64) (replaced []*cmntyp.StandardTransaction) {
	s.LatestHeight = height
	for i, tx := range txs {
		if tx.NativeMessage.Nonce < s.Nonce {
			// Do not insert this transaction into Pool's TxByHash.
			txs[i] = nil
			stat.NumLowNonce++
			continue
		}

		if oldTx, ok := s.TxByNonce[tx.NativeMessage.Nonce]; ok {
			if tx.NativeMessage.GasPrice.Cmp(oldTx.NativeMessage.GasPrice) > 0 {
				// Remove the old transaction from Pool's TxByHash.
				replaced = append(replaced, s.TxByNonce[tx.NativeMessage.Nonce])
				s.TxByNonce[tx.NativeMessage.Nonce] = tx
				stat.NumValid++
			}
		} else {
			s.TxByNonce[tx.NativeMessage.Nonce] = tx
			stat.NumValid++
		}
	}
	return
}

func (s *TxSender) Reap() []*cmntyp.StandardTransaction {
	// Get continuous transactions >= nonce.
	results := make([]*cmntyp.StandardTransaction, 0, len(s.TxByNonce))
	nonce := s.Nonce
	for {
		if tx, ok := s.TxByNonce[nonce]; ok {
			results = append(results, tx)
			nonce++
		} else {
			break
		}
	}

	if len(results) > 0 {
		return results
	}
	return nil
}

func (s *TxSender) Clean(nonce uint64, height uint64) (*TxSender, []*cmntyp.StandardTransaction) {
	if nonce < s.Nonce {
		panic(fmt.Sprintf("[TxSender.Reap] unexpected nonce[%d] got, should >= %d", nonce, s.Nonce))
	}

	deleted := make([]*cmntyp.StandardTransaction, 0, len(s.TxByNonce))
	// Clean outdated transactions.
	if nonce > s.Nonce {
		for i := s.Nonce; i < nonce; i++ {
			if tx, ok := s.TxByNonce[i]; ok {
				deleted = append(deleted, tx)
				delete(s.TxByNonce, i)
			}
		}
		s.Nonce = nonce
	}

	newS := s
	if len(s.TxByNonce) == 0 && (height-s.LatestHeight) >= s.ObsoleteTime {
		newS = nil
	} else if _, ok := s.TxByNonce[s.Nonce]; !ok && (height-s.LatestHeight) >= 2*s.ObsoleteTime {
		newS = nil
		for _, tx := range s.TxByNonce {
			deleted = append(deleted, tx)
		}
	}

	if len(deleted) > 0 {
		return newS, deleted
	}
	return newS, nil
}
