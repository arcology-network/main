package accumulator

import (
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/HPISTechnologies/main/modules/arbitrator/types"
)

const (
	NThreads = 8
)

var transitions []*types.BalanceTransition = make([]*types.BalanceTransition, 0, 1000000)

func CheckBalance(groups [][]*types.ProcessedEuResult) []bool {
	transitions = transitions[:0]
	index := 0
	for i, g := range groups {
		for j, per := range g {
			if per == nil {
				panic(fmt.Sprintf("invalid ProcessedEuResult in group %d, sub group %d\n", i, j))
			}
			for _, t := range per.Transitions {
				t.Tx = uint32(index)
				transitions = append(transitions, t)
			}
			index++
		}
	}
	return Check(transitions, index)
}

func Check(transitions []*types.BalanceTransition, length int) []bool {
	results := make([]bool, length)

	transitionsByPath := make(map[string][]*types.BalanceTransition)
	for i := range transitions {
		transitionsByPath[transitions[i].Path] = append(transitionsByPath[transitions[i].Path], transitions[i])
	}

	accounts := make([]string, 0, len(transitionsByPath))
	for addr := range transitionsByPath {
		accounts = append(accounts, addr)
	}

	var wg sync.WaitGroup
	for i := 0; i < NThreads; i++ {
		wg.Add(1)
		index := i
		go func(index int, from, to int) {
			for j := from; j < to; j++ {
				transitions := transitionsByPath[accounts[j]]
				if len(transitions) == 1 {
					continue
				}

				sort.Slice(transitions[:], func(i, j int) bool {
					return transitions[i].Tx < transitions[j].Tx
				})

				b := transitions[0].Origin
				for _, t := range transitions {
					if sum := new(big.Int).Add(b, t.Delta); sum.Sign() < 0 {
						results[t.Tx] = true
					} else {
						b = sum
					}
				}
			}
			wg.Done()
		}(index, index*len(accounts)/NThreads, (index+1)*len(accounts)/NThreads)
	}
	wg.Wait()

	return results
}
