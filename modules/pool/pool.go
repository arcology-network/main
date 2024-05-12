package pool

import (
	"fmt"

	cmncmn "github.com/arcology-network/common-lib/common"
	ccmap "github.com/arcology-network/common-lib/container/map"
	"github.com/arcology-network/common-lib/exp/mempool"
	cmntyp "github.com/arcology-network/common-lib/types"
	adaptorcommon "github.com/arcology-network/evm-adaptor/pathbuilder"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	apihandler "github.com/arcology-network/evm-adaptor/apihandler"
	statestore "github.com/arcology-network/storage-committer"
	cache "github.com/arcology-network/storage-committer/storage/writecache"
)

type Pool struct {
	ObsoleteTime uint64
	CloseCheck   bool
	TxBySender   *ccmap.ConcurrentMap
	TxByHash     *ccmap.ConcurrentMap
	TxUnchecked  *ccmap.ConcurrentMap
	SourceStat   map[cmntyp.TxSource]*TxSourceStatistics
	StateDB      vm.StateDB

	CherryPickResult []*cmntyp.StandardTransaction
	Waitings         map[evmCommon.Hash]int
	ClearList        []string
}

func NewPool(db *statestore.StateStore, obsoleteTime uint64, closeCheck bool) *Pool {
	api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
		return cache.NewWriteCache(db, 32, 1)
	}, func(cache *cache.WriteCache) { cache.Clear() }))

	return &Pool{
		ObsoleteTime: obsoleteTime,
		CloseCheck:   closeCheck,
		TxBySender:   ccmap.NewConcurrentMap(),
		TxByHash:     ccmap.NewConcurrentMap(),
		TxUnchecked:  ccmap.NewConcurrentMap(),
		SourceStat:   make(map[cmntyp.TxSource]*TxSourceStatistics),
		StateDB:      adaptorcommon.NewImplStateDB(api),
	}

}

func (p *Pool) Add(txs []*cmntyp.StandardTransaction, src cmntyp.TxSource, height uint64) []*cmntyp.StandardTransaction {
	if src.IsForWaitingList() {
		fmt.Printf("[Pool.Add] Receive msgs from %s, len(p.Waitings) = %d\n", src, len(p.Waitings))
		return p.checkWaitingList(txs)
	}

	bySender := make(map[string][]*cmntyp.StandardTransaction)
	uncheckedHashes := make([]string, 0, len(txs))
	uncheckedValues := make([]interface{}, 0, len(txs))
	for i := range txs {
		if !txs[i].NativeMessage.SkipAccountChecks && !p.CloseCheck {
			bySender[string(txs[i].NativeMessage.From.Bytes())] = append(bySender[string(txs[i].NativeMessage.From.Bytes())], txs[i])
		} else {
			uncheckedHashes = append(uncheckedHashes, string(txs[i].TxHash.Bytes()))
			uncheckedValues = append(uncheckedValues, txs[i])
		}
	}

	p.TxUnchecked.BatchSet(uncheckedHashes, uncheckedValues)

	senders := make([]string, 0, len(bySender))
	updates := make([]interface{}, 0, len(bySender))
	replaced := make([][]*cmntyp.StandardTransaction, len(bySender))
	for k, v := range bySender {
		senders = append(senders, k)
		updates = append(updates, v)
	}
	if _, ok := p.SourceStat[src]; !ok {
		p.SourceStat[src] = NewTxSourceStatistics()
	}
	p.TxBySender.BatchUpdate(senders, updates, func(origin interface{}, index int, key string, value interface{}) interface{} {
		var txSender *TxSender
		if origin == nil {
			txSender = NewTxSender(p.StateDB.GetNonce(evmCommon.BytesToAddress([]byte(key))), p.ObsoleteTime)
		} else {
			txSender = origin.(*TxSender)
		}
		replaced[index] = txSender.Add(value.([]*cmntyp.StandardTransaction), p.SourceStat[src], height)
		return txSender
	})

	hashes := uncheckedHashes
	values := uncheckedValues
	for _, u := range updates {
		updated := u.([]*cmntyp.StandardTransaction)
		for i := range updated {
			if updated[i] == nil {
				continue
			}
			hashes = append(hashes, string(updated[i].TxHash.Bytes()))
			values = append(values, updated[i])
		}
	}
	p.TxByHash.BatchSet(hashes, values)

	removed := make([]string, 0, len(txs))
	for _, r := range replaced {
		for i := range r {
			removed = append(removed, string(r[i].TxHash.Bytes()))
		}
	}
	values = make([]interface{}, len(removed))
	p.TxByHash.BatchSet(removed, values)

	return p.checkWaitingList(txs)
}

func (p *Pool) Reap(limit int) []*cmntyp.StandardTransaction {
	shardedResults := p.TxBySender.Traverse(func(key string, value interface{}) (interface{}, interface{}) {
		txSender := value.(*TxSender)
		return value, txSender.Reap()
	})

	results := make([]*cmntyp.StandardTransaction, 0, limit)
	for _, shard := range shardedResults {
		for _, result := range shard {
			txs := result.([]*cmntyp.StandardTransaction)
			results = append(results, txs...)
			if len(results) >= limit {
				return results[:limit]
			}
		}
	}

	if len(results) < limit {
		uncheckedHashes := p.TxUnchecked.Keys()
		uncheckedTxs := p.TxUnchecked.BatchGet(uncheckedHashes[:cmncmn.Min(limit-len(results), len(uncheckedHashes))])
		for _, tx := range uncheckedTxs {
			results = append(results, tx.(*cmntyp.StandardTransaction))
		}
	}
	return results
}

func (p *Pool) QueryByHash(hash evmCommon.Hash) *cmntyp.StandardTransaction {
	keys := make([]string, 1)
	keys[0] = string(hash.Bytes())
	txs := p.TxByHash.BatchGet(keys)
	if txs[0] != nil {
		return txs[0].(*cmntyp.StandardTransaction)
	} else {
		return nil
	}
}

func (p *Pool) CherryPick(hashes []evmCommon.Hash) []*cmntyp.StandardTransaction {
	p.CherryPickResult = make([]*cmntyp.StandardTransaction, len(hashes))
	p.Waitings = make(map[evmCommon.Hash]int)
	keys := make([]string, len(hashes))
	for i, hash := range hashes {
		keys[i] = string(hash.Bytes())
	}

	p.ClearList = keys
	txs := p.TxByHash.BatchGet(keys)
	for i, tx := range txs {
		if tx != nil {
			p.CherryPickResult[i] = tx.(*cmntyp.StandardTransaction)
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
	shardedResults := p.TxBySender.Traverse(func(key string, value interface{}) (interface{}, interface{}) {
		txSender := value.(*TxSender)
		newSender, deleted := txSender.Clean(p.StateDB.GetNonce(evmCommon.BytesToAddress([]byte(key))), height)
		// Cautions: you cannot return *TxSender(nil) as interface{} directly,
		// because *TxSender(nil) != nil.
		if newSender == nil {
			return nil, deleted
		}
		return newSender, deleted
	})

	hashes := make([]string, 0, p.TxByHash.Size())
	for _, shard := range shardedResults {
		for _, result := range shard {
			txs := result.([]*cmntyp.StandardTransaction)
			for _, tx := range txs {
				hashes = append(hashes, string(tx.TxHash.Bytes()))
			}
		}
	}
	// Use the default value nil to delete all the entries.
	values := make([]interface{}, len(hashes))
	p.TxByHash.BatchSet(hashes, values)

	values = make([]interface{}, len(p.ClearList))
	p.TxByHash.BatchSet(p.ClearList, values)
	p.TxUnchecked.BatchSet(p.ClearList, values)
}

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
