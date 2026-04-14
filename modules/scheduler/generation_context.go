package scheduler

import (
	"context"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/scheduler/conflictor"
	workload "github.com/arcology-network/scheduler/workload"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type generationContext struct {
	genID      int
	generation *generation

	remaining []*workload.JobSequence

	// ===== exec phase =====
	execIssued   map[int]bool
	execReceived map[int][]*mtypes.JobSequenceResponse
	execPending  map[int][]*workload.JobSequence

	execResponses map[evmCommon.Hash]*mtypes.ExecuteResponse
	newContracts  []evmCommon.Address
	executed      []evmCommon.Hash

	// ===== arb phase =====
	arbIssued bool
	arbDone   bool
	cpRight   []uint64

	finished bool
}

func NewGenerationContext(generation *generation, Id int, sequences []*workload.JobSequence) *generationContext {
	return &generationContext{
		genID:      Id,
		generation: generation,
		remaining:  sequences,

		execIssued:    map[int]bool{},
		execReceived:  map[int][]*mtypes.JobSequenceResponse{},
		execPending:   map[int][]*workload.JobSequence{},
		execResponses: map[evmCommon.Hash]*mtypes.ExecuteResponse{},
	}
}
func (gc *generationContext) execIssue(execId int, cap int, batchSize int) (requests []*workload.JobSequence, finish bool) {
	singleTx := 0
	ColllectIdx := -1
	batches := 0
	for i := range gc.remaining {
		jobSize := len(gc.remaining[i].Jobs)

		if singleTx > 0 && batches == cap-1 {
			if jobSize > 1 {
				ColllectIdx = i - 1
				break
			}
		}

		if jobSize == 1 {
			singleTx++
			if singleTx >= batchSize {
				batches++
				singleTx = 0
			}
		} else {
			batches++
		}
		if batches == cap {
			if i > 0 {
				ColllectIdx = i - 1
			} else {
				ColllectIdx = i
			}
			break
		}
		ColllectIdx = i
	}

	if ColllectIdx == len(gc.remaining)-1 {
		requests = gc.remaining
		gc.remaining = gc.remaining[:0]
		finish = true
	} else {
		requests = gc.remaining[:ColllectIdx+1]
		gc.remaining = gc.remaining[ColllectIdx+1:] //clear
		finish = false
	}

	// if cap >= len(gc.remaining) {
	// 	//all
	// 	requests = gc.remaining
	// 	gc.remaining = gc.remaining[:0] //clear
	// 	finish = true
	// } else {
	// 	//part
	// 	requests = gc.remaining[:cap]
	// 	gc.remaining = gc.remaining[cap:] //clear
	// 	finish = false
	// }
	gc.execPending[execId] = append(gc.execPending[execId], requests...)
	gc.execIssued[execId] = true
	return
}
func (gc *generationContext) onExecResult(resp *mtypes.ExecResponses) {
	execId := int(resp.ExecId)
	gc.execReceived[execId] = append(gc.execReceived[execId], resp.Resp...)
}
func (gc *generationContext) isExecCompleted() bool {
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
	responses := make(map[evmCommon.Hash]*mtypes.ExecuteResponse, mtypes.MaxBlockSize)
	contractAddress := make([]evmCommon.Address, 0, mtypes.MaxBlockSize)
	executed := make([]evmCommon.Hash, 0, mtypes.MaxBlockSize)
	for execId := range gc.execIssued {
		for _, resps := range gc.execReceived[execId] {
			for k := range resps.Responses {
				responses[resps.Responses[k].Hash] = resps.Responses[k]
				executed = append(executed, resps.Responses[k].Hash)
			}
			contractAddress = append(contractAddress, resps.ContractAddress...)
		}
	}
	gc.execResponses = responses
	gc.newContracts = contractAddress
	gc.executed = executed
	logger.Log.Debug(context.Background(), "gc.executed", "CollectExecResults", logger.F("gc.executed", len(gc.executed)), logger.F("gc.newContracts", len(gc.newContracts)))
}

func (gc *generationContext) onArbitrateResult(resp *conflictor.CollisionSummary) {
	gc.arbDone = true
	for _, collision := range resp.Collisions {
		for _, peer := range collision.Peers {
			gc.cpRight = append(gc.cpRight, peer.JobID)
		}
	}
	gc.finished = true
}
