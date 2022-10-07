package scheduler

import (
	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
)

type dataPool struct {
	list []*ethCommon.Hash
}

func newDataPool() *dataPool {
	return &dataPool{
		list: make([]*ethCommon.Hash, 0),
	}
}

func (pool *dataPool) addData(list []*ethCommon.Hash) {
	pool.list = append(pool.list, list...)
}

func (pool *dataPool) getData() []*ethCommon.Hash {
	return pool.list
}

func (pool *dataPool) clear() {
	pool.list = pool.list[:0]
}
