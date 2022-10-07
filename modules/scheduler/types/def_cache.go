package types

import (
	"sync"

	"github.com/HPISTechnologies/common-lib/types"
)

type DefCache struct {
	defs map[string]*[]*types.ExecuteResponse
	lock sync.RWMutex
}

func NewDefCache() *DefCache {
	return &DefCache{
		defs: map[string]*[]*types.ExecuteResponse{},
	}
}
func (dc *DefCache) Add(result *types.ExecuteResponse) {
	dc.lock.Lock()
	defer dc.lock.Unlock()

	//gather arbitrate list
	if result.DfCall == nil {
		return
	}
	defid := result.DfCall.DeferID

	resultList := dc.defs[defid]
	if resultList == nil {
		resultList = &[]*types.ExecuteResponse{}
		dc.defs[defid] = resultList
	}
	*resultList = append(*resultList, result)
}

func (dc *DefCache) GetAll() map[string]*[]*types.ExecuteResponse {
	dc.lock.Lock()
	defer dc.lock.Unlock()
	return dc.defs
}
