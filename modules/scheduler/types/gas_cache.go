package types

import (
	"sort"
	"sync"

	mtypes "github.com/arcology-network/main/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type GasCache struct {
	DictionaryHash map[evmCommon.Hash]uint64
	lock           sync.RWMutex
}

func (gc *GasCache) CostCalculateSort(txElements *[][][]*mtypes.TxElement) {
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
				if gasused, ok := gc.DictionaryHash[*element.TxHash]; ok {
					costs[i].cost = costs[i].cost + gasused
				}

			}
		}

		costItems := CostItems(costs)
		sort.Sort(costItems)

		sortedList := make([][]*mtypes.TxElement, len(firstDemension))
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
