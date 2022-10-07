package types

import (
	"fmt"

	"github.com/HPISTechnologies/common-lib/types"
)

type ExectimeCaches struct {
	caches *DataCache
}

func NewExectimeCaches(cache int) *ExectimeCaches {
	return &ExectimeCaches{
		caches: NewDataCache(cache),
	}
}
func (ec *ExectimeCaches) GetKey(height uint64) string {
	return fmt.Sprintf("statisticalInformation-%v", height)
}

func (ec *ExectimeCaches) Query(height uint64) *types.StatisticalInformation {
	key := ec.GetKey(height)
	staticalInfo := ec.caches.Query(key)
	if staticalInfo != nil {
		return staticalInfo.(*types.StatisticalInformation)
	}
	return nil
}

func (ec *ExectimeCaches) Save(height uint64, staticalInfo *types.StatisticalInformation) {
	key := ec.GetKey(height)
	ec.caches.Add(height, []string{key}, []interface{}{staticalInfo})
}
