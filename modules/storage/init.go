package storage

import (
	"encoding/gob"

	"github.com/arcology-network/common-lib/transactional"
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
)

func init() {
	gob.Register(&SchdState{})
	gob.Register([]SchdState{})

	actor.Factory.Register("storage", NewStorage)
	actor.Factory.Register("storage_debug", NewStorageDebug)
	actor.Factory.Register("storage.initializer", NewInitializer)
	actor.Factory.Register("storage.root_calculator", NewRootCalculator)
	actor.Factory.Register("storage.metrics", NewMetrics)
	actor.Factory.Register("storage.statesyncstore", NewStateSyncStore)
	actor.Factory.Register("storage.schdstore", NewSchdStore)
	actor.Factory.Register("storage.cacheblockstore", NewCacheBlockStore)

	intf.Factory.Register("storage.tmblockstore", func(int, string) interface{} {
		return NewTmBlockStore()
	})
	intf.Factory.Register("storage.tmstatestore", func(int, string) interface{} {
		return NewTmStateStore()
	})
	intf.Factory.Register("storage.urlstore", func(int, string) interface{} {
		return NewUrlStore()
	})
	// intf.Factory.Register("storage.txstore", func(int, string) interface{} {
	// 	return NewTxStore()
	// })
	intf.Factory.Register("storage.receiptstore", func(int, string) interface{} {
		return NewReceiptStore()
	})
	intf.Factory.Register("storage.blockstore", func(int, string) interface{} {
		return NewBlockStore()
	})
	intf.Factory.Register("storage.statestore", func(int, string) interface{} {
		return NewStateStore()
	})
	intf.Factory.Register("storage.debugstore", func(int, string) interface{} {
		return NewDebugStore()
	})
	intf.Factory.Register("storage.indexerstore", func(int, string) interface{} {
		return NewIndexerStore()
	})
	intf.Factory.Register("storage.statesyncstore", func(concurrency int, groupId string) interface{} {
		return NewStateSyncStore(concurrency, groupId)
	})
	intf.Factory.Register("storage", func(concurrency int, groupId string) interface{} {
		return NewStorage(concurrency, groupId)
	})
	intf.Factory.Register("storage.transactionalstore", func(int, string) interface{} {
		return transactional.NewTransactionalStore()
	})
	intf.Factory.Register("storage.schdstore", func(concurrency int, groupId string) interface{} {
		return NewSchdStore(concurrency, groupId)
	})

}
