package types

import (
	"fmt"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/main/modules/scheduler/lib"
)

type ExecutingSchedule struct {
	//Uuid      uint64
	Batches   [][]*types.ExecutingSequence
	batchIdx  int
	scheduler *lib.Scheduler
}

func NewExecutingSchedule(scheduler *lib.Scheduler) *ExecutingSchedule {
	return &ExecutingSchedule{
		Batches:   [][]*types.ExecutingSequence{},
		batchIdx:  -1,
		scheduler: scheduler,
	}
}

func (execSched *ExecutingSchedule) Reset(stdMsgs []*types.StandardMessage, height uint64) (string, int, uint32) {
	sequencess, err := execSched.scheduler.Schedule(stdMsgs, height)
	if len(err) > 0 {
		fmt.Printf("start errlog=%v\n", err)
	}
	start := uint32(1)
	for i := range sequencess {
		for j := range sequencess[i] {
			start = InitTxIds(sequencess[i][j], start)
		}
	}

	execSched.Batches = sequencess
	execSched.batchIdx = -1
	return "", len(sequencess), start
}

func (execSched *ExecutingSchedule) GetNextBatch() []*types.ExecutingSequence {
	if execSched.batchIdx+1 < len(execSched.Batches) {
		execSched.batchIdx++
		return execSched.Batches[execSched.batchIdx]
	}
	return []*types.ExecutingSequence{}
}

func (execSched *ExecutingSchedule) Append(list []*ConflictListItem) string {
	logs := ""
	for _, item := range list {
		ret := execSched.scheduler.Update([]*ethCommon.Address{item.Left}, []*ethCommon.Address{item.Right})
		logs += ret
	}
	return logs
}

func (execSched *ExecutingSchedule) Init(conflictFile string) string {
	logs := ""
	conflictList, err := LoadingConf(conflictFile)
	if err != nil {
		logs += fmt.Sprintf("loading conf err=%v\n", err)
		return logs
	}
	for _, item := range conflictList {
		lefts := make([]*ethCommon.Address, len(item.Left))
		for i, entrance := range item.Left {
			addr := ethCommon.HexToAddress(entrance.ContractAddress)
			lefts[i] = &addr
		}
		rights := make([]*ethCommon.Address, len(item.Right))
		for i, entrance := range item.Right {
			addr := ethCommon.HexToAddress(entrance.ContractAddress)
			rights[i] = &addr
		}
		ret := execSched.scheduler.Update(lefts, rights)
		logs += ret
	}
	return logs

}
