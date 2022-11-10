package types

import (
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
)

type ExeclogCaches struct {
	caches      *DataCache
	concurrency int
}

func NewExeclogCaches(cache int, concurrency int) *ExeclogCaches {
	return &ExeclogCaches{
		caches:      NewDataCache(cache),
		concurrency: concurrency,
	}
}

func (ec *ExeclogCaches) Query(key string) *string {
	execlog := ec.caches.Query(key)
	if execlog != nil {
		return execlog.(*string)
	}
	return nil
}

func (ec *ExeclogCaches) Save(height uint64, execlogs []string) {
	if len(execlogs) == 0 {
		return
	}
	keys := make([]string, len(execlogs))
	datas := make([]interface{}, len(execlogs))
	worker := func(start, end int, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			logs := types.ExecutingLogs{}
			err := logs.UnMarshal(execlogs[i])
			if err == nil {
				keys[i] = string(logs.Txhash.Bytes())
				datas[i] = &execlogs[i]
			}
		}
	}
	common.ParallelWorker(len(execlogs), ec.concurrency, worker)
	ec.caches.Add(height, keys, datas)
}
