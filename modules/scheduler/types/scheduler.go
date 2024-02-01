package types

import (
	"fmt"

	"github.com/arcology-network/main/modules/scheduler/lib"
	mtypes "github.com/arcology-network/main/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type ExecutingSchedule struct {
	Batches   [][]*mtypes.ExecutingSequence
	batchIdx  int
	scheduler *lib.Scheduler
}

func NewExecutingSchedule(scheduler *lib.Scheduler) *ExecutingSchedule {
	return &ExecutingSchedule{
		Batches:   [][]*mtypes.ExecutingSequence{},
		batchIdx:  -1,
		scheduler: scheduler,
	}
}

func (execSched *ExecutingSchedule) Init(conflictFile string) string {
	logs := ""
	conflictList, err := LoadingConf(conflictFile)
	if err != nil {
		logs += fmt.Sprintf("loading conf err=%v\n", err)
		return logs
	}
	for _, item := range conflictList {
		lefts := make([]*evmCommon.Address, len(item.Left))
		for i, entrance := range item.Left {
			addr := evmCommon.HexToAddress(entrance.ContractAddress)
			lefts[i] = &addr
		}
		rights := make([]*evmCommon.Address, len(item.Right))
		for i, entrance := range item.Right {
			addr := evmCommon.HexToAddress(entrance.ContractAddress)
			rights[i] = &addr
		}
		ret := execSched.scheduler.Update(lefts, rights)
		logs += ret
	}
	return logs

}
