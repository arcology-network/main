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

package storage

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"

	kafkalib "github.com/arcology-network/streamer/kafka/lib"
	"github.com/arcology-network/streamer/logger"
)

type CheckedListItem struct {
	createTime time.Time
	tx         []byte
	from       byte
}

type CheckedList struct {
	all             map[[32]byte]CheckedListItem
	lock            sync.RWMutex
	totals          uint64
	hits            uint64
	timeoutForClear time.Duration
}

// NewList returns a new CheckedList structure.
func NewCheckList(waits int64) *CheckedList {
	cl := CheckedList{
		all:             make(map[[32]byte]CheckedListItem),
		totals:          0,
		hits:            0,
		timeoutForClear: time.Duration(waits) * time.Second,
	}
	tim := kafkalib.SyncTimer{}
	tim.StartTimer(cl.timeoutForClear, cl.timerClear)
	return &cl
}

func (t *CheckedList) timerClear() {
	t.lock.Lock()
	defer t.lock.Unlock()

	clearCounter := 0
	for k, v := range t.all {
		if v.createTime.Add(t.timeoutForClear).Before(time.Now()) {
			delete(t.all, k)
			clearCounter++
		}
	}
	logger.Log.Debug(context.Background(), "CheckedList", "checklist hit rate", logger.F("checked", t.totals), logger.F("hit", t.hits))
}

func (t *CheckedList) ExistTx(tx []byte, from byte) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.totals = t.totals + 1
	hash := sha256.Sum256(tx)
	_, ok := t.all[hash]
	if !ok {
		t.all[hash] = CheckedListItem{
			createTime: time.Now(),
			tx:         tx,
			from:       from,
		}
		return false
	}
	t.hits = t.hits + 1
	return true
}
