package storage

import (
	"github.com/arcology-network/common-lib/common"
	univaluepk "github.com/arcology-network/concurrenturl/univalue"
	eushared "github.com/arcology-network/eu/shared"
)

func GetTransitionIds(euresults []*eushared.EuResult) []uint32 {
	txIds := make([]uint32, len(euresults))
	for i, euresult := range euresults {
		txIds[i] = euresult.ID
	}
	return txIds
}

func GetTransitions(euresults []*eushared.EuResult) ([]uint32, []*univaluepk.Univalue) {
	txIds := make([]uint32, len(euresults))
	transitionsize := 0
	for i, euresult := range euresults {
		transitionsize = transitionsize + len(euresult.Transitions)
		txIds[i] = euresult.ID
	}
	threadNum := 6
	transitionses := make([][]*univaluepk.Univalue, threadNum)
	worker := func(start, end, index int, args ...interface{}) {
		for i := start; i < end; i++ {
			transitionses[index] = append(transitionses[index], univaluepk.Univalues{}.Decode(euresults[i].Transitions).(univaluepk.Univalues)...)
		}
	}
	common.ParallelWorker(len(euresults), threadNum, worker)

	transitions := make([]*univaluepk.Univalue, 0, transitionsize)
	for _, trans := range transitionses {
		transitions = append(transitions, trans...)
	}
	return txIds, transitions
}
