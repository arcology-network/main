package types

import (
	"sync"
)

const (
	KeySize = 10
)

type DataCache struct {
	Data map[string]interface{}

	DataHeights []uint64
	Indexer     map[uint64][]string
	CacheSize   int

	lock sync.RWMutex
}

func NewDataCache(size int) *DataCache {
	return &DataCache{
		CacheSize:   size,
		Data:        map[string]interface{}{},
		DataHeights: []uint64{},
		Indexer:     map[uint64][]string{},
	}
}
func (caches *DataCache) Query(hash string) interface{} {
	caches.lock.Lock()
	defer caches.lock.Unlock()

	if data, ok := caches.Data[hash]; ok {
		return data
	}

	return nil
}

func (caches *DataCache) QueryBlock(height uint64) []interface{} {
	caches.lock.Lock()
	defer caches.lock.Unlock()

	data := []interface{}{}
	if hashes, ok := caches.Indexer[height]; ok {
		for i := range hashes {
			if v, ok := caches.Data[hashes[i]]; ok {
				data = append(data, v)
			}
		}
		if len(hashes) == len(data) {
			return data
		}
	}

	return nil
}

func (caches *DataCache) GetHashes(height uint64) []string {
	caches.lock.Lock()
	defer caches.lock.Unlock()

	if hashes, ok := caches.Indexer[height]; ok {
		return hashes
	}

	return []string{}
}

func (caches *DataCache) Add(height uint64, keys []string, data []interface{}) {
	caches.lock.Lock()
	defer caches.lock.Unlock()

	if len(keys) == 0 {
		return
	}

	hashes, ok := caches.Indexer[height]
	if ok {
		hashes = append(hashes, keys...)
	} else {
		hashes = keys
		caches.DataHeights = append(caches.DataHeights, height)
	}
	for i, key := range keys {
		caches.Data[key] = data[i]
	}
	caches.Indexer[height] = hashes

	if len(caches.DataHeights) > caches.CacheSize {
		removeKeys := caches.Indexer[caches.DataHeights[0]]
		delete(caches.Indexer, caches.DataHeights[0])
		caches.DataHeights = caches.DataHeights[1:]
		for _, key := range removeKeys {
			delete(caches.Data, key)
		}
	}
}
