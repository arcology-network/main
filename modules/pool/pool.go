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

	cmncmn "github.com/arcology-network/common-lib/common"
	ccmap "github.com/arcology-network/common-lib/exp/map"
	"github.com/arcology-network/common-lib/exp/mempool"
	cmntyp "github.com/arcology-network/common-lib/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	apihandler "github.com/arcology-network/eu/apihandler"

	ethimpl "github.com/arcology-network/eu/ethadaptor"
	statecache "github.com/arcology-network/state-engine/state/cache"
)

func stringHasher(k string) uint64 {
	var hash uint64
	for i := 0; i < len(k); i++ {
		hash += uint64(k[i])
	}
	return hash % 16
}

type Pool struct {
	ObsoleteTime uint64
	CloseCheck   bool
	// TxBySender   *ccmap.ConcurrentMap
	// TxByHash     *ccmap.ConcurrentMap
	// TxUnchecked  *ccmap.ConcurrentMap

	TxBySender  *ccmap.ConcurrentMap[string, *TxSender]
	TxByHash    *ccmap.ConcurrentMap[string, *cmntyp.StandardTransaction]
	TxUnchecked *ccmap.ConcurrentMap[string, *cmntyp.StandardTransaction]

	SourceStat map[cmntyp.TxSource]*TxSourceStatistics
	StateDB    vm.StateDB

	CherryPickResult []*cmntyp.StandardTransaction
	Waitings         map[evmCommon.Hash]int
	ClearList        []string
}

func NewPool(db *statecache.ExecutionStateStore, obsoleteTime uint64, closeCheck bool) *Pool {
	ccRuntime := apihandler.NewConcurrentRuntime(
		0, // concurrent runtime ID
		mempool.NewMempool(
			16,
			1,
			func() *statecache.ExecutionStateStore {
				// When creating a new writecache, use store as the backend.
				return statecache.NewExecutionStateStore(db, 32, 1)
			},
			func(cache *statecache.ExecutionStateStore) {
				cache.Clear()
			}),
	)

	stateDB := ethimpl.NewImplStateDB(ccRuntime)

	return &Pool{
		ObsoleteTime: obsoleteTime,
		CloseCheck:   closeCheck,
		TxBySender: ccmap.NewConcurrentMap[string, *TxSender](
			16,
			func(v *TxSender) bool { return v == nil },
			stringHasher,
		),

		TxByHash: ccmap.NewConcurrentMap[string, *cmntyp.StandardTransaction](
			16,
			func(v *cmntyp.StandardTransaction) bool { return v == nil },
			stringHasher,
		),

		TxUnchecked: ccmap.NewConcurrentMap[string, *cmntyp.StandardTransaction](
			16,
			func(v *cmntyp.StandardTransaction) bool { return v == nil },
			stringHasher,
		),
		SourceStat: make(map[cmntyp.TxSource]*TxSourceStatistics),
		StateDB:    stateDB,
	}

}

func (p *Pool) Add(txs []*cmntyp.StandardTransaction, src cmntyp.TxSource, height uint64) []*cmntyp.StandardTransaction {
	if src.IsForWaitingList() {
		fmt.Printf("[Pool.Add] Receive msgs from %s, len(p.Waitings) = %d\n", src, len(p.Waitings))
		return p.checkWaitingList(txs)
	}

	bySender := make(map[string][]*cmntyp.StandardTransaction)
	uncheckedHashes := make([]string, 0, len(txs))
	uncheckedValues := make([]*cmntyp.StandardTransaction, 0, len(txs))
	for i := range txs {
		if !txs[i].NativeMessage.SkipNonceChecks && !p.CloseCheck {
			bySender[string(txs[i].NativeMessage.From.Bytes())] = append(bySender[string(txs[i].NativeMessage.From.Bytes())], txs[i])
		} else {
			uncheckedHashes = append(uncheckedHashes, string(txs[i].TxHash.Bytes()))
			uncheckedValues = append(uncheckedValues, txs[i])
		}
	}

	p.TxUnchecked.SetBatch(uncheckedHashes, uncheckedValues)

	senders := make([]string, 0, len(bySender))
	updates := make([][]*cmntyp.StandardTransaction, 0, len(bySender))
	replaced := make([][]*cmntyp.StandardTransaction, len(bySender))
	for k, v := range bySender {
		senders = append(senders, k)
		updates = append(updates, v)
	}
	if _, ok := p.SourceStat[src]; !ok {
		p.SourceStat[src] = NewTxSourceStatistics()
	}

	for sender, txs := range bySender {
		key := sender

		origin, _ := p.TxBySender.Get(key)
		txSender := origin
		if txSender == nil {
			txSender = NewTxSender(
				p.StateDB.GetNonce(evmCommon.BytesToAddress([]byte(key))),
				p.ObsoleteTime,
			)
		}

		replacedTxs := txSender.Add(txs, p.SourceStat[src], height)
		replaced = append(replaced, replacedTxs)

		p.TxBySender.Set(key, txSender)
	}

	hashes := uncheckedHashes
	values := uncheckedValues
	for _, u := range updates {
		updated := u
		for i := range updated {
			if updated[i] == nil {
				continue
			}
			hashes = append(hashes, string(updated[i].TxHash.Bytes()))
			values = append(values, updated[i])
		}
	}
	p.TxByHash.SetBatch(hashes, values)

	removed := make([]string, 0, len(txs))
	for _, r := range replaced {
		for i := range r {
			removed = append(removed, string(r[i].TxHash.Bytes()))
		}
	}
	values = make([]*cmntyp.StandardTransaction, len(removed))
	p.TxByHash.SetBatch(removed, values)

	return p.checkWaitingList(txs)
}

func (p *Pool) Reap(limit int) []*cmntyp.StandardTransaction {
	results := make([]*cmntyp.StandardTransaction, 0, limit)

	p.TxBySender.ParallelForeachDo(func(_ string, sender *TxSender) {
		if len(results) >= limit {
			return
		}
		txs := sender.Reap()
		results = append(results, txs...)
	})

	if len(results) >= limit {
		return results[:limit]
	}

	// unchecked
	keys := p.TxUnchecked.Keys()
	if len(keys) > 0 {
		values, found := p.TxUnchecked.GetBatch(
			keys[:cmncmn.Min(limit-len(results), len(keys))],
		)
		for i, ok := range found {
			if ok {
				results = append(results, values[i])
			}
		}
	}
	return results
}

func (p *Pool) QueryByHash(hash evmCommon.Hash) *cmntyp.StandardTransaction {
	keys := make([]string, 1)
	// keys[0] = string(hash.Bytes())
	// txs := p.TxByHash.BatchGet(keys)
	// if txs[0] != nil {
	// 	return txs[0].(*cmntyp.StandardTransaction)
	// } else {
	// 	return nil
	// }

	values, found := p.TxByHash.GetBatch(keys)
	if found[0] {
		return values[0]
	}
	return nil
}

func (p *Pool) CherryPick(hashes []evmCommon.Hash) []*cmntyp.StandardTransaction {
	p.CherryPickResult = make([]*cmntyp.StandardTransaction, len(hashes))
	p.Waitings = make(map[evmCommon.Hash]int)
	keys := make([]string, len(hashes))
	for i, hash := range hashes {
		keys[i] = string(hash.Bytes())
	}

	p.ClearList = keys
	txs, _ := p.TxByHash.GetBatch(keys)
	for i, tx := range txs {
		if tx != nil {
			p.CherryPickResult[i] = tx
		} else {
			p.Waitings[hashes[i]] = i
		}
	}

	if len(p.Waitings) == 0 {
		result := p.CherryPickResult
		p.CherryPickResult = nil
		p.Waitings = nil
		return result
	}
	return nil
}

func (p *Pool) Clean(height uint64) {
	deletedCh := make(chan *cmntyp.StandardTransaction, 1024)

	p.TxBySender.Traverse(func(key string, sender **TxSender) {
		if *sender == nil {
			return
		}

		newSender, deleted := (*sender).Clean(
			p.StateDB.GetNonce(evmCommon.BytesToAddress([]byte(key))),
			height,
		)

		for _, tx := range deleted {
			deletedCh <- tx
		}

		if newSender == nil {
			*sender = nil // isNilVal(nil) => delete
		} else {
			*sender = newSender
		}
	})

	close(deletedCh)

	hashes := make([]string, 0)
	for tx := range deletedCh {
		hashes = append(hashes, string(tx.TxHash.Bytes()))
	}

	if len(hashes) > 0 {
		values := make([]*cmntyp.StandardTransaction, len(hashes))
		p.TxByHash.SetBatch(hashes, values)
	}

	if len(p.ClearList) > 0 {
		values := make([]*cmntyp.StandardTransaction, len(p.ClearList))
		p.TxByHash.SetBatch(p.ClearList, values)
		p.TxUnchecked.SetBatch(p.ClearList, values)
	}
}

// func (p *Pool) Clean(height uint64) {
// 	shardedResults := p.TxBySender.Traverse(func(key string, value interface{}) (interface{}, interface{}) {
// 		txSender := value.(*TxSender)
// 		newSender, deleted := txSender.Clean(p.StateDB.GetNonce(evmCommon.BytesToAddress([]byte(key))), height)
// 		// Cautions: you cannot return *TxSender(nil) as interface{} directly,
// 		// because *TxSender(nil) != nil.
// 		if newSender == nil {
// 			return nil, deleted
// 		}
// 		return newSender, deleted
// 	})

// 	hashes := make([]string, 0, p.TxByHash.Length())
// 	for _, shard := range shardedResults {
// 		for _, result := range shard {
// 			txs := result.([]*cmntyp.StandardTransaction)
// 			for _, tx := range txs {
// 				hashes = append(hashes, string(tx.TxHash.Bytes()))
// 			}
// 		}
// 	}
// 	// Use the default value nil to delete all the entries.
// 	values := make([]*cmntyp.StandardTransaction, len(hashes))
// 	p.TxByHash.BatchSet(hashes, values)

// 	values = make([]*cmntyp.StandardTransaction, len(p.ClearList))
// 	p.TxByHash.BatchSet(p.ClearList, values)
// 	p.TxUnchecked.BatchSet(p.ClearList, values)
// }

func (p *Pool) checkWaitingList(txs []*cmntyp.StandardTransaction) []*cmntyp.StandardTransaction {
	if len(p.Waitings) > 0 {
		for _, tx := range txs {
			if index, ok := p.Waitings[tx.TxHash]; ok {
				p.CherryPickResult[index] = tx
				delete(p.Waitings, tx.TxHash)
				if len(p.Waitings) == 0 {
					result := p.CherryPickResult
					p.CherryPickResult = nil
					p.Waitings = nil
					fmt.Printf("[Pool.checkWaitingList] Waiting list fulfilled.")
					return result
				}
			}
		}
		fmt.Printf("[Pool.checkWaitingList] len(p.Waitings) = %d\n", len(p.Waitings))
	}
	return nil
}
