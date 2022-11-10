package pool

import (
	"fmt"

	cmntyp "github.com/arcology-network/common-lib/types"
)

type TxSender struct {
	Nonce        uint64
	TxByNonce    map[uint64]*cmntyp.StandardMessage
	ObsoleteTime uint64
	LatestHeight uint64
}

func NewTxSender(nonce uint64, obsoleteTime uint64) *TxSender {
	return &TxSender{
		Nonce:        nonce,
		TxByNonce:    make(map[uint64]*cmntyp.StandardMessage),
		ObsoleteTime: obsoleteTime,
	}
}

func (s *TxSender) Add(txs []*cmntyp.StandardMessage, stat *TxSourceStatistics, height uint64) (replaced []*cmntyp.StandardMessage) {
	s.LatestHeight = height
	for i, tx := range txs {
		if tx.Native.Nonce() < s.Nonce {
			// Do not insert this transaction into Pool's TxByHash.
			txs[i] = nil
			stat.NumLowNonce++
			continue
		}

		if oldTx, ok := s.TxByNonce[tx.Native.Nonce()]; ok {
			if tx.Native.GasPrice().Cmp(oldTx.Native.GasPrice()) > 0 {
				// Remove the old transaction from Pool's TxByHash.
				replaced = append(replaced, s.TxByNonce[tx.Native.Nonce()])
				s.TxByNonce[tx.Native.Nonce()] = tx
				stat.NumValid++
			}
		} else {
			s.TxByNonce[tx.Native.Nonce()] = tx
			stat.NumValid++
		}
	}
	return
}

func (s *TxSender) Reap() []*cmntyp.StandardMessage {
	// Get continuous transactions >= nonce.
	results := make([]*cmntyp.StandardMessage, 0, len(s.TxByNonce))
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

func (s *TxSender) Clean(nonce uint64, height uint64) (*TxSender, []*cmntyp.StandardMessage) {
	if nonce < s.Nonce {
		panic(fmt.Sprintf("[TxSender.Reap] unexpected nonce[%d] got, should >= %d", nonce, s.Nonce))
	}

	deleted := make([]*cmntyp.StandardMessage, 0, len(s.TxByNonce))
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
