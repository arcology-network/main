package types

import (
	"sort"
	"sync"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/types"
)

type GasCache struct {
	dictionaryHash map[ethCommon.Hash]uint64
	lock           sync.RWMutex
}

func NewGasCache() *GasCache {
	return &GasCache{
		dictionaryHash: map[ethCommon.Hash]uint64{},
		lock:           sync.RWMutex{},
	}
}
func (gc *GasCache) Add(hash ethCommon.Hash, gas uint64) {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	gc.dictionaryHash[hash] = gas
}

func (gc *GasCache) Clear(hash ethCommon.Hash, gas uint64) {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	gc.dictionaryHash = map[ethCommon.Hash]uint64{}
}

func (gc *GasCache) CostCalculateSort(txElements *[][][]*types.TxElement) {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	if txElements == nil {
		return
	}

	for idx, firstDemension := range *txElements {
		costs := make([]CostItem, len(firstDemension))
		for i, secondDemension := range firstDemension {
			costs[i].idx = i
			for _, element := range secondDemension {
				if gasused, ok := gc.dictionaryHash[*element.TxHash]; ok {
					costs[i].cost = costs[i].cost + gasused
				}

			}
		}

		costItems := CostItems(costs)
		sort.Sort(costItems)

		sortedList := make([][]*types.TxElement, len(firstDemension))
		for i, item := range costItems {
			sortedList[i] = firstDemension[item.idx]
		}
		(*txElements)[idx] = sortedList
	}

}

type CostItem struct {
	cost uint64
	idx  int
}

type CostItems []CostItem

func (cis CostItems) Len() int           { return len(cis) }
func (cis CostItems) Swap(i, j int)      { cis[i], cis[j] = cis[j], cis[i] }
func (cis CostItems) Less(i, j int) bool { return cis[i].cost > cis[j].cost }
