package pool

import (
	"fmt"

	cmncmn "github.com/arcology-network/common-lib/common"
	ccmap "github.com/arcology-network/common-lib/container/map"
	cmntyp "github.com/arcology-network/common-lib/types"
	url "github.com/arcology-network/concurrenturl"
	"github.com/arcology-network/concurrenturl/interfaces"
	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/evm/core/vm"
	ccapi "github.com/arcology-network/vm-adaptor/api"
	"github.com/arcology-network/vm-adaptor/eth"
)

type Pool struct {
	ObsoleteTime uint64
	CloseCheck   bool
	TxBySender   *ccmap.ConcurrentMap
	TxByHash     *ccmap.ConcurrentMap
	TxUnchecked  *ccmap.ConcurrentMap
	SourceStat   map[cmntyp.TxSource]*TxSourceStatistics
	StateDB      vm.StateDB

	CherryPickResult []*cmntyp.StandardMessage
	Waitings         map[evmCommon.Hash]int
	ClearList        []string
}

func NewPool(db interfaces.Datastore, obsoleteTime uint64, closeCheck bool) *Pool {

	url := url.NewConcurrentUrl(db)
	api := ccapi.NewAPI(url)

	return &Pool{
		ObsoleteTime: obsoleteTime,
		CloseCheck:   closeCheck,
		TxBySender:   ccmap.NewConcurrentMap(),
		TxByHash:     ccmap.NewConcurrentMap(),
		TxUnchecked:  ccmap.NewConcurrentMap(),
		SourceStat:   make(map[cmntyp.TxSource]*TxSourceStatistics),
		StateDB:      eth.NewImplStateDB(api), // adaptor.NewStateDBV2(nil, db, url.NewConcurrentUrl(db)),
	}

}

func (p *Pool) Add(txs []*cmntyp.StandardMessage, src cmntyp.TxSource, height uint64) []*cmntyp.StandardMessage {
	if src.IsForWaitingList() {
		fmt.Printf("[Pool.Add] Receive msgs from %s, len(p.Waitings) = %d\n", src, len(p.Waitings))
		return p.checkWaitingList(txs)
	}

	bySender := make(map[string][]*cmntyp.StandardMessage)
	uncheckedHashes := make([]string, 0, len(txs))
	uncheckedValues := make([]interface{}, 0, len(txs))
	for i := range txs {
		if !txs[i].Native.SkipAccountChecks && !p.CloseCheck {
			bySender[string(txs[i].Native.From.Bytes())] = append(bySender[string(txs[i].Native.From.Bytes())], txs[i])
		} else {
			uncheckedHashes = append(uncheckedHashes, string(txs[i].TxHash.Bytes()))
			uncheckedValues = append(uncheckedValues, txs[i])
		}
	}

	p.TxUnchecked.BatchSet(uncheckedHashes, uncheckedValues)

	senders := make([]string, 0, len(bySender))
	updates := make([]interface{}, 0, len(bySender))
	replaced := make([][]*cmntyp.StandardMessage, len(bySender))
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
		replaced[index] = txSender.Add(value.([]*cmntyp.StandardMessage), p.SourceStat[src], height)
		return txSender
	})

	hashes := uncheckedHashes
	values := uncheckedValues
	for _, u := range updates {
		updated := u.([]*cmntyp.StandardMessage)
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

func (p *Pool) Reap(limit int) []*cmntyp.StandardMessage {
	shardedResults := p.TxBySender.Traverse(func(key string, value interface{}) (interface{}, interface{}) {
		txSender := value.(*TxSender)
		return value, txSender.Reap()
	})

	results := make([]*cmntyp.StandardMessage, 0, limit)
	for _, shard := range shardedResults {
		for _, result := range shard {
			txs := result.([]*cmntyp.StandardMessage)
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
			results = append(results, tx.(*cmntyp.StandardMessage))
		}
	}
	return results
}

func (p *Pool) QueryByHash(hash evmCommon.Hash) *cmntyp.StandardMessage {
	keys := make([]string, 1)
	keys[0] = string(hash.Bytes())
	txs := p.TxByHash.BatchGet(keys)
	if txs[0] != nil {
		return txs[0].(*cmntyp.StandardMessage)
	} else {
		return nil
	}
}

func (p *Pool) CherryPick(hashes []evmCommon.Hash) []*cmntyp.StandardMessage {
	p.CherryPickResult = make([]*cmntyp.StandardMessage, len(hashes))
	p.Waitings = make(map[evmCommon.Hash]int)
	keys := make([]string, len(hashes))
	for i, hash := range hashes {
		keys[i] = string(hash.Bytes())
	}

	p.ClearList = keys
	txs := p.TxByHash.BatchGet(keys)
	for i, tx := range txs {
		if tx != nil {
			p.CherryPickResult[i] = tx.(*cmntyp.StandardMessage)
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
			txs := result.([]*cmntyp.StandardMessage)
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

func (p *Pool) checkWaitingList(txs []*cmntyp.StandardMessage) []*cmntyp.StandardMessage {
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
