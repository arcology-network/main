package types

import (
	"sort"
	"sync"

	evmCommon "github.com/ethereum/go-ethereum/common"
)

type GasCache struct {
	DictionaryHash map[evmCommon.Hash]uint64
	lock           sync.RWMutex
}

func (gc *GasCache) CostCalculateSort(txElements [][]evmCommon.Hash) [][]evmCommon.Hash {
	gc.lock.Lock()
	defer gc.lock.Unlock()

	if len(txElements) == 0 {
		return txElements
	}

	costs := make([]CostItem, len(txElements))
	for i, firstDemension := range txElements {
		costs[i].idx = i
		for _, element := range firstDemension {
			if gasused, ok := gc.DictionaryHash[element]; ok {
				costs[i].cost = costs[i].cost + gasused
			}
		}
	}
	costItems := CostItems(costs)
	sort.Sort(costItems)

	sortedList := make([][]evmCommon.Hash, len(txElements))
	for i, item := range costItems {
		sortedList[i] = txElements[item.idx]
	}
	return sortedList
}

type CostItem struct {
	cost uint64
	idx  int
}

type CostItems []CostItem

func (cis CostItems) Len() int           { return len(cis) }
func (cis CostItems) Swap(i, j int)      { cis[i], cis[j] = cis[j], cis[i] }
func (cis CostItems) Less(i, j int) bool { return cis[i].cost > cis[j].cost }
