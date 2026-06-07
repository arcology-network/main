/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package storage

import (
	"github.com/arcology-network/common-lib/common"
	statecell "github.com/arcology-network/common-lib/crdt/statecell"
	eushared "github.com/arcology-network/eu/shared"

	statecache "github.com/arcology-network/state-engine/state/cache"
	statecommitter "github.com/arcology-network/state-engine/state/committer"
)

type InitAsyncObj struct {
	StateStore *statecache.ExecutionStateStore
	Committer  *statecommitter.StateCommitter
}

func GetTransitionIds(euresults []*eushared.EuResult) []uint64 {
	txIds := make([]uint64, len(euresults))
	for i, euresult := range euresults {
		txIds[i] = euresult.ID
	}
	return txIds
}

func GetTransitions(euresults []*eushared.EuResult) ([]uint64, []*statecell.StateCell) {
	txIds := make([]uint64, len(euresults))
	transitionsize := 0
	for i, euresult := range euresults {
		transitionsize = transitionsize + len(euresult.Trans)
		txIds[i] = euresult.ID
	}
	threadNum := 6
	transitionses := make([][]*statecell.StateCell, threadNum)
	worker := func(start, end, index int, args ...interface{}) {
		for i := start; i < end; i++ {
			transitionses[index] = append(transitionses[index], euresults[i].Trans...)
		}
	}
	common.ParallelWorker(len(euresults), threadNum, worker)

	transitions := make([]*statecell.StateCell, 0, transitionsize)
	for _, trans := range transitionses {
		transitions = append(transitions, trans...)
	}
	return txIds, transitions
}
