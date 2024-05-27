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
	eushared "github.com/arcology-network/eu/shared"
	univaluepk "github.com/arcology-network/storage-committer/univalue"
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
		transitionsize = transitionsize + len(euresult.Trans)
		txIds[i] = euresult.ID
	}
	threadNum := 6
	transitionses := make([][]*univaluepk.Univalue, threadNum)
	worker := func(start, end, index int, args ...interface{}) {
		for i := start; i < end; i++ {
			transitionses[index] = append(transitionses[index], euresults[i].Trans...)
		}
	}
	common.ParallelWorker(len(euresults), threadNum, worker)

	transitions := make([]*univaluepk.Univalue, 0, transitionsize)
	for _, trans := range transitionses {
		transitions = append(transitions, trans...)
	}
	return txIds, transitions
}
