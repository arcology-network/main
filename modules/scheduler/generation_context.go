package scheduler

import (
	"context"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type generationContext struct {
	genID      int
	generation *generation

	remaining []*mtypes.ExecutorRequest

	// ===== exec phase =====
	execIssued   map[int]bool                        // Executor idx 已经发起执行
	execReceived map[int][]*mtypes.ExecutorResponses // 每台 Executor 已返回的结果
	execPending  map[int][]*mtypes.ExecutorRequest   // 每台 Executor 待执行队列

	execResponses map[evmCommon.Hash]*mtypes.ExecuteResponse
	newContracts  []evmCommon.Address
	executed      []evmCommon.Hash

	// ===== arb phase =====
	arbIssued bool
	arbDone   bool
	cpLeft    []uint64
	cpRight   []uint64

	finished bool
}

func NewGenerationContext(generation *generation, Id int, requests []*mtypes.ExecutorRequest) *generationContext {
	return &generationContext{
		genID:      Id,
		generation: generation,
		remaining:  requests,

		execIssued:    map[int]bool{},
		execReceived:  map[int][]*mtypes.ExecutorResponses{},
		execPending:   map[int][]*mtypes.ExecutorRequest{},
		execResponses: map[evmCommon.Hash]*mtypes.ExecuteResponse{},
	}
}
func (gc *generationContext) execIssue(execId int, cap int) (requests []*mtypes.ExecutorRequest, finish bool) {
	if cap >= len(gc.remaining) {
		//all
		requests = gc.remaining
		gc.remaining = gc.remaining[:0] //clear
		finish = true
	} else {
		//part
		requests = gc.remaining[:cap]
		gc.remaining = gc.remaining[cap:] //clear
		finish = false
	}
	gc.execPending[execId] = append(gc.execPending[execId], requests...)
	gc.execIssued[execId] = true
	return
}
func (gc *generationContext) onExecResult(resp *mtypes.ExecutorResponses) {
	execId := int(resp.ExecId)
	gc.execReceived[execId] = append(gc.execReceived[execId], resp)
}
func (gc *generationContext) isExecCompleted() bool {
	// gc := g.context.generationCtx[g.context.currentGenerationID]
	if len(gc.remaining) > 0 {
		return false
	}
	for execId := range gc.execIssued {
		if len(gc.execPending[execId]) != len(gc.execReceived[execId]) {
			return false
		}
	}
	return true
}
func (gc *generationContext) CollectExecResults() {
	responses := make(map[evmCommon.Hash]*mtypes.ExecuteResponse, maxBlockSize)
	contractAddress := make([]evmCommon.Address, 0, maxBlockSize)
	executed := make([]evmCommon.Hash, 0, maxBlockSize)
	for execId := range gc.execIssued {
		for _, resps := range gc.execReceived[execId] {
			for k := range resps.HashList {
				responses[resps.HashList[k]] = &mtypes.ExecuteResponse{
					Hash:    resps.HashList[k],
					Status:  resps.StatusList[k],
					GasUsed: resps.GasUsedList[k],
				}
				executed = append(executed, resps.HashList[k])
			}
			contractAddress = append(contractAddress, resps.ContractAddresses...)
		}
	}
	gc.execResponses = responses
	gc.newContracts = contractAddress
	gc.executed = executed
	logger.Log.Debug(context.Background(), "gc.executed", "CollectExecResults", logger.F("gc.executed", len(gc.executed)), logger.F("gc.newContracts", len(gc.newContracts)))
}

func (gc *generationContext) onArbitrateResult(resp *mtypes.ArbitratorResponse) {
	gc.arbDone = true
	gc.cpLeft = resp.CPairLeft
	gc.cpRight = resp.CPairRight
	gc.finished = true
}
