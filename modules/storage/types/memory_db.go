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
	"errors"
	"sync"

	"github.com/arcology-network/common-lib/common"
	ccctrnmap "github.com/arcology-network/common-lib/container/map"
)

type MemoryDB struct {
	mutex sync.RWMutex
	db    *ccctrnmap.ConcurrentMap
}

func NewMemoryDB() *MemoryDB {
	return &MemoryDB{
		db: ccctrnmap.NewConcurrentMap(),
	}
}

func (this *MemoryDB) Set(key string, value []byte) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	this.db.Set(key, value)
	return nil
}

func (this *MemoryDB) Get(key string) ([]byte, error) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	result, _ := this.db.Get(key)
	if result != nil {
		return result.([]byte), nil
	}
	return []byte{}, errors.New("Error: Key not found: " + key)
}

func (this *MemoryDB) BatchGet(keys []string) ([][]byte, error) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	objs := this.db.BatchGet(keys)
	counter := len(keys)
	results := make([][]byte, counter)
	errs := make([]error, counter)

	getter := func(start, end, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			if objs[i] != nil {
				results[i] = objs[i].([]byte)
			} else {
				errs[i] = errors.New("Error: Key not found: " + keys[i])
			}
		}
	}
	common.ParallelWorker(counter, nthread, getter)

	for _, err := range errs {
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

func (this *MemoryDB) BatchSet(keys []string, data [][]byte) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	objs := make([]interface{}, len(data))
	for i, data := range data {
		objs[i] = data
	}
	this.db.BatchSet(keys, objs)
	return nil
}
func (this *MemoryDB) Query(pattern string, condition func(string, string) bool) ([]string, [][]byte, error) {
	return []string{}, [][]byte{}, nil
}
