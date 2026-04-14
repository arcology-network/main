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
	cmap "github.com/arcology-network/common-lib/exp/map"
	types "github.com/arcology-network/common-lib/types"
	schtyp "github.com/arcology-network/main/modules/scheduler/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/scheduler/conflictor"
	"github.com/arcology-network/scheduler/workload"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type generation struct {
	context *processContext
	gen     *workload.Generation
}

func (g *generation) startProcess(
	ctx *actor.ExecutionContext,
) {
	gc := g.CurrentContext()
	gc.generation.context.executor.StartIssue(ctx, gc)
}

func (g *generation) NextProcess(
	ctx *actor.ExecutionContext,
	execId int,
) bool {
	gc := g.CurrentContext()
	return gc.generation.context.executor.Issue(ctx, gc, execId)
}

func countMsg(gc *generationContext) int {
	counter := 0
	for i := range gc.generation.gen.JobSeqs {
		counter += len(gc.generation.gen.JobSeqs[i].Jobs)
	}
	return counter
}

func (g *generation) OnExecResult(
	resp *mtypes.ExecResponses,
) bool {
	gc := g.CurrentContext()
	gc.onExecResult(resp)
	return gc.isExecCompleted()
}

func (g *generation) StartArbitrate(
	ctx *actor.ExecutionContext,
) {
	gc := g.CurrentContext()
	gc.CollectExecResults()
	list := g.makeArbitrateParam(gc.execResponses)
	g.context.arbitrator.Issue(ctx, list)
	gc.arbIssued = true
}

func (g *generation) CurrentContext() *generationContext {
	return g.context.generationCtx[g.context.currentGenerationID]
}

func newGeneration(context *processContext, gen *workload.Generation) *generation {
	for _, seq := range gen.JobSeqs {
		for i := range seq.Jobs {
			seq.Jobs[i].ID = uint64(context.txId)
			context.txId++

			context.txHash2IdBiMap.Add(seq.Jobs[i].StdMsg.TxHash, seq.Jobs[i].ID)
		}
	}
	return &generation{
		context: context,
		gen:     gen,
	}
}

func (g *generation) CollectGenerationResult() *types.InclusiveList {
	gc := g.context.GetCurrentGeneration().CurrentContext()
	flags := make([]bool, len(gc.executed))

	currentGenList := make([]evmCommon.Hash, 0, len(gc.executed))
	for i := range g.gen.JobSeqs {
		for j := range g.gen.JobSeqs[i].Jobs {
			g.context.executed = append(g.context.executed, g.gen.JobSeqs[i].Jobs[j].StdMsg.TxHash)
			currentGenList = append(currentGenList, g.gen.JobSeqs[i].Jobs[j].StdMsg.TxHash)
		}
	}

	g.context.newContracts = append(g.context.newContracts, gc.newContracts...)

	deletedDict := make(map[evmCommon.Hash]struct{})
	for _, rId := range gc.cpRight {
		deletedDict[g.context.txHash2IdBiMap.GetInverse(rId)] = struct{}{}
	}

	for i, hash := range currentGenList {
		if _, ok := deletedDict[hash]; !ok {
			flags[i] = true
		}
	}

	g.context.deletedDict = cmap.Merge(g.context.deletedDict, deletedDict)

	nextIdx := gc.genID + 1
	if g.context.generationCount == nextIdx {
		nextIdx = 0
	}

	return &types.InclusiveList{
		HashList:          currentGenList,
		Successful:        flags,
		NextGenerationIdx: uint32(nextIdx),
	}
}

func (g *generation) makeArbitrateParam(
	responses map[evmCommon.Hash]*mtypes.ExecuteResponse,
) [][]evmCommon.Hash {
	arbitrateParam := make([][]evmCommon.Hash, 0, len(g.gen.JobSeqs))
	for i := range g.gen.JobSeqs {
		for j := range g.gen.JobSeqs[i].Jobs {
			arbitrateParam = append(arbitrateParam, []evmCommon.Hash{g.gen.JobSeqs[i].Jobs[j].StdMsg.TxHash})
		}
	}
	for h, response := range responses {
		g.context.txHash2Gas[h] = response.GasUsed
	}
	return (&schtyp.GasCache{DictionaryHash: g.context.txHash2Gas}).CostCalculateSort(arbitrateParam)
}

// 1. Update `context.txHash2Callee` for each message;
// 2. Update `context.txHash2Sign` for each message;
// 3. Collect parallel messages' hash, put them into `executed` and return.
func (g *generation) setMsgProperty() {
	for seqIdx, _ := range g.gen.JobSeqs {
		g.gen.JobSeqs[seqIdx].ID = uint64(seqIdx)
	}
}
func (g *generation) onArbitrateResult(ctx *actor.ExecutionContext, resp *conflictor.CollisionSummary) {
	gc := g.CurrentContext()
	gc.onArbitrateResult(resp)
}
