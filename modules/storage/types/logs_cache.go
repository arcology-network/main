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
	"sync"

	mtypes "github.com/arcology-network/main/types"
	evm "github.com/ethereum/go-ethereum"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type LogCaches struct {
	Caches       []*mtypes.LogCache
	CacheSize    int
	LatestHeight uint64
	lock         sync.RWMutex
}

func NewLogCaches(size int) *LogCaches {
	return &LogCaches{
		CacheSize:    size,
		LatestHeight: 0,
		Caches:       make([]*mtypes.LogCache, 0, size),
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
	return mtypes.FilteLogs(logs, filter)
}

func (lc *LogCaches) Add(height uint64, receipts []*evmTypes.Receipt) {
	lc.lock.Lock()
	defer lc.lock.Unlock()
	logs := mtypes.ToLogs(receipts)
	if len(logs) == 0 {
		return
	}
	cache := mtypes.LogCache{
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
