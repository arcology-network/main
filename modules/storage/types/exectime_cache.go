package types

import (
	"fmt"

	mtypes "github.com/arcology-network/main/types"
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

func (ec *ExectimeCaches) Query(height uint64) *mtypes.StatisticalInformation {
	key := ec.GetKey(height)
	staticalInfo := ec.caches.Query(key)
	if staticalInfo != nil {
		return staticalInfo.(*mtypes.StatisticalInformation)
	}
	return nil
}

func (ec *ExectimeCaches) Save(height uint64, staticalInfo *mtypes.StatisticalInformation) {
	key := ec.GetKey(height)
	ec.caches.Add(height, []string{key}, []interface{}{staticalInfo})
}
