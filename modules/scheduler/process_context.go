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

	cmncmn "github.com/arcology-network/common-lib/common"
	evmCommon "github.com/ethereum/go-ethereum/common"

	mtypes "github.com/arcology-network/main/types"
)

type processContext struct {
	generationCtx map[int]*generationContext
	height        uint64

	executor   *ExecClient
	arbitrator *RpcClientArbitrate

	// Per block data.
	timestamp *big.Int

	txHash2IdBiMap *cmncmn.BiMap[evmCommon.Hash, uint64]
	txHash2Gas     map[evmCommon.Hash]uint64

	txId                uint32
	currentGenerationID int
	generationCount     int

	// Results collected for scheduler.
	newContracts []evmCommon.Address
	conflicts    *mtypes.ConflictInfos

	executed    []evmCommon.Hash
	deletedDict map[evmCommon.Hash]struct{}
}

func (c *processContext) GetCurrentGeneration() *generation {
	return c.generationCtx[c.currentGenerationID].generation
}

func createProcessContext() *processContext {
	return &processContext{
		conflicts: mtypes.NewConflictInfos(),
		txId:      1,
	}
}

func (c *processContext) init(execBatchSize int, executors []*mtypes.ExecutorConf) {
	c.executor = NewExecClient(execBatchSize, executors)
	c.arbitrator = NewRpcClientArbitrate()
}

func (c *processContext) onNewBlock(height uint64) {
	c.txHash2IdBiMap = cmncmn.NewBiMap[evmCommon.Hash, uint64]()
	c.txHash2Gas = make(map[evmCommon.Hash]uint64)
	c.executed = c.executed[:0]
	c.deletedDict = make(map[evmCommon.Hash]struct{})
	c.txId = 1
	c.currentGenerationID = -1
	c.newContracts = c.newContracts[:0]
	c.conflicts.Reset()
	c.height = height
}

func (c *processContext) onStartBlock(gens []*generation) {
	c.generationCount = len(gens)
	c.generationCtx = make(map[int]*generationContext, len(gens))
	for i := range gens {
		gens[i].setMsgProperty()
		c.generationCtx[i] = NewGenerationContext(gens[i], i, gens[i].gen.JobSeqs)
	}
}

func (c *processContext) onNewGeneration() {
	c.currentGenerationID++
}
