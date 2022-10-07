package types

import (
	"github.com/HPISTechnologies/common-lib/common"
	"github.com/HPISTechnologies/common-lib/mhasher"
	types "github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/component-lib/log"
	"go.uber.org/zap"
)

type ExecutionSchedule struct {
	Uuid     uint64
	Batches  [][]*types.StandardMessage
	batchIdx int
}

func NewExecutionSchedule(stdMsgs []*types.StandardMessage, concurrencyLookup *map[string]int, inlog *actor.WorkerThreadLogger) *ExecutionSchedule {
	indices, err := mhasher.SortByHash(types.StandardMessages(stdMsgs).Hashes())

	if err != nil {
		inlog.Log(log.LogLevel_Error, "NewExecutionSchedule err", zap.String("err", err.Error()))
	} else {
		sortedMsgs := make([]*types.StandardMessage, len(stdMsgs))

		worker := func(start, end, idx int, args ...interface{}) {
			sortings := args[0].([]interface{})[0].([]*types.StandardMessage)
			idxs := args[0].([]interface{})[1].([]uint64)
			soreted := args[0].([]interface{})[2].([]*types.StandardMessage)

			for i := start; i < end; i++ {
				soreted[i] = sortings[int(idxs[i])]
			}
		}
		common.ParallelWorker(len(stdMsgs), 4, worker, stdMsgs, indices, sortedMsgs)
		stdMsgs = sortedMsgs

	}

	serials := make([][]*types.StandardMessage, 0, len(stdMsgs))
	parallels := make([]*types.StandardMessage, 0, len(stdMsgs))

	flags := CheckParallel(stdMsgs, concurrencyLookup)

	for i := range stdMsgs {
		if flags[i] {
			parallels = append(parallels, stdMsgs[i])
		} else {
			serials = append(serials, []*types.StandardMessage{stdMsgs[i]})
		}
	}

	return &ExecutionSchedule{
		Uuid:     common.GenerateUUID(),
		Batches:  append(serials, parallels),
		batchIdx: -1,
	}
}

func CheckParallel(stdMsgs []*types.StandardMessage, concurrencyLookup *map[string]int) []bool {
	flags := make([]bool, len(stdMsgs))
	worker := func(start int, end int, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			signer := stdMsgs[i].Native.EntrySignature()
			if signer == "" {
				flags[i] = true
				continue
			}
			if concurrency, ok := (*concurrencyLookup)[stdMsgs[i].Native.EntrySignature()]; ok && concurrency > 1 {
				flags[i] = true
			}
		}
	}
	common.ParallelWorker(len(stdMsgs), 64, worker)
	return flags
}

func (execSched *ExecutionSchedule) GetNextBatch() []*types.StandardMessage {
	if execSched.batchIdx+1 < len(execSched.Batches) {
		execSched.batchIdx++
		return execSched.Batches[execSched.batchIdx]
	}
	return []*types.StandardMessage{}
}
