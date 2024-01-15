package storage

import (
	"sync"

	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

var (
	cbsSingleton actor.IWorkerEx
	initCbs      sync.Once
)

type CacheBlockStore struct {
	actor.WorkerThread
}

func NewCacheBlockStore(concurrency int, groupId string) actor.IWorkerEx {
	initCbs.Do(func() {
		cbsSingleton = &CacheBlockStore{}
		cbsSingleton.(*CacheBlockStore).Set(concurrency, groupId)
	})
	return cbsSingleton
}

func (cbs *CacheBlockStore) Inputs() ([]string, bool) {
	return []string{actor.MsgPendingBlock}, false
}

func (cbs *CacheBlockStore) Outputs() map[string]int {
	return map[string]int{}
}

func (cbs *CacheBlockStore) OnStart() {}

func (cbs *CacheBlockStore) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch msg.Name {
	case actor.MsgPendingBlock:
		block := msg.Data.(*cmntyp.MonacoBlock)
		var na int
		intf.Router.Call("blockstore", "SavePendingBlock", block, &na)
	}
	return nil
}
