package backend

import (
	"fmt"
	"sync"
	"time"

	evmCommon "github.com/arcology-network/evm/common"

	"github.com/arcology-network/component-lib/ethrpc"
	eth "github.com/arcology-network/evm"
	ethcmn "github.com/arcology-network/evm/common"
	ethtyp "github.com/arcology-network/evm/core/types"
)

const (
	FilterTypeLogs byte = iota
	FilterTypeBlock
	FilterTypePendingTransaction
)

type Filter struct {
	Typ      byte
	Deadline *time.Timer // filter is inactiv when deadline triggers
	Hashes   []ethcmn.Hash
	Crit     eth.FilterQuery
	Logs     []*ethtyp.Log
	lock     sync.Mutex
}

func (f *Filter) getHashes() []ethcmn.Hash {
	f.lock.Lock()
	defer f.lock.Unlock()
	hashes := f.Hashes
	f.Hashes = nil
	return hashes
}

func (f *Filter) getLogs() []*ethtyp.Log {
	f.lock.Lock()
	defer f.lock.Unlock()
	logs := f.Logs
	f.Logs = nil
	return logs
}

func (f *Filter) append(height uint64, logs []*ethtyp.Log, blockhash evmCommon.Hash) {
	f.lock.Lock()
	defer f.lock.Unlock()

	switch f.Typ {
	case FilterTypeLogs:
		filteredLogs := make([]*ethtyp.Log, 0, len(logs))
		if f.Crit.BlockHash != nil {
			if *f.Crit.BlockHash == ethcmn.Hash(blockhash) {
				filteredLogs = logs
			}
		} else {
			found := true
			if f.Crit.FromBlock != nil && height < f.Crit.FromBlock.Uint64() {
				found = false
			}
			if found && f.Crit.ToBlock != nil && height > f.Crit.ToBlock.Uint64() {
				found = false
			}
			if found {
				filteredLogs = logs
			}
		}
		finalLogs := ethrpc.FilteLogs(filteredLogs, f.Crit)
		f.Logs = append(f.Logs, finalLogs...)
	case FilterTypeBlock:
		f.Hashes = append(f.Hashes, ethcmn.BytesToHash(blockhash[:]))
	}
}

type Filters struct {
	filtersMu sync.Mutex
	filters   map[ID]*Filter
	timeout   time.Duration
	//backend   internal.EthereumAPI
}

var (
	filtersSingleton *Filters
	initOnce         sync.Once
)

func NewFilters() *Filters {
	initOnce.Do(func() {
		filtersSingleton = &Filters{
			filters: make(map[ID]*Filter),
		}
	})
	return filtersSingleton
}

func (fs *Filters) SetTimeout(timeout time.Duration) {
	fs.timeout = timeout
	go fs.timeoutLoop(timeout)
}

func (fs *Filters) OnResultsArrived(height uint64, receipts []*ethtyp.Receipt, blockhash evmCommon.Hash) {
	logs := ethrpc.ToLogs(receipts)
	for _, f := range fs.filters {
		go f.append(height, logs, blockhash)
	}
}

// timeoutLoop runs at the interval set by 'timeout' and deletes filters
// that have not been recently used. It is started when the API is created.
func (fs *Filters) timeoutLoop(timeout time.Duration) {
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		<-ticker.C
		fs.filtersMu.Lock()
		for id, f := range fs.filters {
			select {
			case <-f.Deadline.C:
				delete(fs.filters, id)
			default:
				continue
			}
		}
		fs.filtersMu.Unlock()
	}
}

func (fs *Filters) UninstallFilter(id ID) bool {
	fs.filtersMu.Lock()
	defer fs.filtersMu.Unlock()

	_, found := fs.filters[id]
	if found {
		delete(fs.filters, id)
	}
	return found
}

func (fs *Filters) NewPendingTransactionFilter() ID {
	fs.filtersMu.Lock()
	defer fs.filtersMu.Unlock()

	id := NewID()
	fs.filters[id] = &Filter{
		Typ:      FilterTypePendingTransaction,
		Deadline: time.NewTimer(fs.timeout),
		Hashes:   make([]ethcmn.Hash, 0),
	}
	return id
}

func (fs *Filters) NewBlockFilter() ID {
	fs.filtersMu.Lock()
	defer fs.filtersMu.Unlock()

	id := NewID()
	fs.filters[id] = &Filter{
		Typ:      FilterTypeBlock,
		Deadline: time.NewTimer(fs.timeout),
		Hashes:   make([]ethcmn.Hash, 0),
	}

	return id
}

func (fs *Filters) NewFilter(crit eth.FilterQuery) ID {
	fs.filtersMu.Lock()
	defer fs.filtersMu.Unlock()

	id := NewID()
	fs.filters[id] = &Filter{
		Typ:      FilterTypeLogs,
		Crit:     crit,
		Deadline: time.NewTimer(fs.timeout),
		Logs:     make([]*ethtyp.Log, 0),
	}
	return id
}

func (fs *Filters) GetFilterChanges(id ID) (interface{}, error) {
	fs.filtersMu.Lock()
	defer fs.filtersMu.Unlock()

	if f, found := fs.filters[id]; found {
		if !f.Deadline.Stop() {
			// timer expired but filter is not yet removed in timeout loop
			// receive timer value and reset timer
			<-f.Deadline.C
		}
		f.Deadline.Reset(fs.timeout)

		switch f.Typ {
		case FilterTypePendingTransaction, FilterTypeBlock:
			hashes := f.getHashes()
			return returnHashes(hashes), nil
		case FilterTypeLogs:
			logs := f.getLogs()
			return returnLogs(logs), nil
		}
	}

	return []interface{}{}, fmt.Errorf("filter not found")
}

func (fs *Filters) GetFilterLogsCrit(id ID) (*eth.FilterQuery, error) {
	fs.filtersMu.Lock()
	f, found := fs.filters[id]
	fs.filtersMu.Unlock()

	if !found || f.Typ != FilterTypeLogs {
		return nil, fmt.Errorf("filter not found")
	}
	return &f.Crit, nil
	// logs, err := fs.backend.GetLogs(f.Crit)
	// if err != nil {
	// 	return nil, err
	// }
	// return returnLogs(logs), nil
}

func returnLogs(logs []*ethtyp.Log) []*ethtyp.Log {
	if logs == nil {
		return []*ethtyp.Log{}
	}
	return logs
}

func returnHashes(hashes []ethcmn.Hash) []ethcmn.Hash {
	if hashes == nil {
		return []ethcmn.Hash{}
	}
	return hashes
}
