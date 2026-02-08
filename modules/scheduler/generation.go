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
	"context"

	"github.com/arcology-network/common-lib/common"
	types "github.com/arcology-network/common-lib/types"
	schtyp "github.com/arcology-network/main/modules/scheduler/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type generation struct {
	context   *processContext
	sequences []*mtypes.ExecutingSequence
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

// func (g *generation) isExecCompleted() bool {
// 	gc := g.CurrentContext()
// 	return gc.isExecCompleted()
// }

func (g *generation) OnExecResult(
	resp *mtypes.ExecutorResponses,
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

func newGeneration(context *processContext, sequences []*mtypes.ExecutingSequence) *generation {
	for _, seq := range sequences {
		for i := range seq.Msgs {
			seq.Msgs[i].ID = uint64(context.txId)
			context.txId++

			context.txHash2IdBiMap.Add(seq.Msgs[i].TxHash, seq.Msgs[i].ID)
		}
	}
	return &generation{
		context:   context,
		sequences: sequences,
	}
}

func (g *generation) CollectGenerationResult() *types.InclusiveList {
	gc := g.context.GetCurrentGeneration().CurrentContext()
	flags := make([]bool, len(gc.executed))

	g.context.executed = append(g.context.executed, gc.executed...)
	g.context.newContracts = append(g.context.newContracts, gc.newContracts...)

	for i := range gc.cpLeft {
		ltxhash := g.context.txHash2IdBiMap.GetInverse(uint64(gc.cpLeft[i]))
		leftAddr, ok := g.context.txHash2Callee[ltxhash]
		if !ok {
			continue
		}
		leftSign, ok := g.context.txHash2Sign[ltxhash]
		if !ok {
			continue
		}

		rtxhash := g.context.txHash2IdBiMap.GetInverse(uint64(gc.cpRight[i]))
		rightAddr, ok := g.context.txHash2Callee[rtxhash]
		if !ok {
			continue
		}
		rightSign, ok := g.context.txHash2Sign[rtxhash]
		if !ok {
			continue
		}
		g.context.conflicts.Add(&mtypes.ConflictInfo{
			LeftAddress:  leftAddr,
			LeftSign:     leftSign,
			RightAddress: rightAddr,
			RightSign:    rightSign,
		})
	}

	deletedDict := make(map[evmCommon.Hash]struct{})
	for _, rId := range gc.cpRight {
		deletedDict[g.context.txHash2IdBiMap.GetInverse(rId)] = struct{}{}
	}

	for i, hash := range gc.executed {
		if _, ok := deletedDict[hash]; !ok {
			flags[i] = true
		}
	}

	common.MergeMaps(g.context.deletedDict, deletedDict)
	logger.Log.Debug(context.Background(), "gc.executed", "CollectGenerationResult", logger.F("gc.executed", gc.executed))
	return &types.InclusiveList{
		HashList:   gc.executed,
		Successful: flags,
	}
}

func (g *generation) makeArbitrateParam(
	responses map[evmCommon.Hash]*mtypes.ExecuteResponse,
) [][]evmCommon.Hash {
	arbitrateParam := make([][]evmCommon.Hash, 0, len(g.sequences))
	for i := range g.sequences {
		if g.sequences[i].Parallel {
			for j := range g.sequences[i].Msgs {
				arbitrateParam = append(arbitrateParam, []evmCommon.Hash{g.sequences[i].Msgs[j].TxHash})
			}
		} else {
			hashes := make([]evmCommon.Hash, 0, len(g.sequences[i].Msgs))
			for j := range g.sequences[i].Msgs {
				hashes = append(hashes, g.sequences[i].Msgs[j].TxHash)
			}
			arbitrateParam = append(arbitrateParam, hashes)
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
	// executed := make([]evmCommon.Hash, 0, 50000)

	for _, seq := range g.sequences {
		for _, msg := range seq.Msgs {
			if msg.Native.To != nil {
				g.context.txHash2Callee[msg.TxHash] = *msg.Native.To
				sign := msg.Native.Data
				if len(msg.Native.Data) > 4 {
					sign = sign[:4]
				}
				if len(sign) == 0 {
					g.context.txHash2Sign[msg.TxHash] = [4]byte{}
				} else {
					g.context.txHash2Sign[msg.TxHash] = [4]byte(sign)
				}
			}
		}
	}
}
func (g *generation) onArbitrateResult(ctx *actor.ExecutionContext, resp *mtypes.ArbitratorResponse) {
	gc := g.CurrentContext()
	gc.onArbitrateResult(resp)

	// logger.Log.Debug(context.Background(), "gc.executed 1", "generation.onArbitrateResult", logger.F("gc.executed", g.context.executed))
	// g.CollectGenerationResult()
	// logger.Log.Debug(context.Background(), "gc.executed 2", "generation.onArbitrateResult", logger.F("gc.executed", g.context.executed))
}
