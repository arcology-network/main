package types

import (
	"sync"

	ethTypes "github.com/arcology-network/3rd-party/eth/types"
	"github.com/arcology-network/component-lib/ethrpc"
	evm "github.com/arcology-network/evm"
	evmCommon "github.com/arcology-network/evm/common"
	evmTypes "github.com/arcology-network/evm/core/types"
)

type LogCaches struct {
	Caches       []*ethrpc.LogCache
	CacheSize    int
	LatestHeight uint64
	lock         sync.RWMutex
}

func NewLogCaches(size int) *LogCaches {
	return &LogCaches{
		CacheSize:    size,
		LatestHeight: 0,
		Caches:       make([]*ethrpc.LogCache, 0, size),
	}
}

func (lc *LogCaches) Query(filter evm.FilterQuery) []*evmTypes.Log {
	lc.lock.Lock()
	defer lc.lock.Unlock()
	logs := []*evmTypes.Log{}
	if filter.BlockHash != nil {
		for _, cache := range lc.Caches {
			if cache.BlockHash == *filter.BlockHash {
				logs = cache.Logs
			}
		}
	} else {
		from := lc.LatestHeight
		if filter.FromBlock != nil {
			from = filter.FromBlock.Uint64()
		}
		end := lc.LatestHeight
		if filter.ToBlock != nil {
			end = filter.ToBlock.Uint64()
		}
		if end > lc.LatestHeight {
			end = lc.LatestHeight
		}
		firstHeight := uint64(0)
		if len(lc.Caches) > 0 {
			firstHeight = lc.Caches[0].Height
		}
		if from < firstHeight {
			from = firstHeight
		}
		for _, cache := range lc.Caches {
			if cache.Height >= from && cache.Height <= end {
				logs = append(logs, cache.Logs...)
			}
		}
	}
	return ethrpc.FilteLogs(logs, filter)
}

func (lc *LogCaches) Add(height uint64, receipts []*ethTypes.Receipt) {
	lc.lock.Lock()
	defer lc.lock.Unlock()
	logs := ethrpc.ToLogs(receipts)
	if len(logs) == 0 {
		return
	}
	cache := ethrpc.LogCache{
		Logs:      logs,
		Height:    height,
		BlockHash: evmCommon.Hash(receipts[0].BlockHash),
	}

	lc.Caches = append(lc.Caches, &cache)
	if len(lc.Caches) > lc.CacheSize {
		lc.Caches = lc.Caches[1:]
	}
	lc.LatestHeight = height
}
