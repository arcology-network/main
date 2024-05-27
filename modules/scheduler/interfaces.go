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

package scheduler

import (
	"math/big"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type Arbitrator interface {
	Start()
	Stop()
	Do(els [][]evmCommon.Hash, log *actor.WorkerThreadLogger, generationIdx int) ([]evmCommon.Hash, []uint32, []uint32)
}

type Executor interface {
	Start()
	Stop()
	Run(sequences []*mtypes.ExecutingSequence, timestamp *big.Int, msgTemplate *actor.Message, inlog *actor.WorkerThreadLogger, parallelism int, height uint64, generationIdx int) (map[evmCommon.Hash]*mtypes.ExecuteResponse, []evmCommon.Address)
}
