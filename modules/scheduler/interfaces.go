package scheduler

import (
	"math/big"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	schedulingTypes "github.com/HPISTechnologies/main/modules/scheduler/types"
)

type Arbitrator interface {
	Start()
	Stop()
	Do(els [][][]*types.TxElement, log *actor.WorkerThreadLogger, generationIdx int, batchIdx int) ([]*ethCommon.Hash, []uint32, []uint32)
}

type Executor interface {
	Start()
	Stop()
	Run(msgs map[ethCommon.Hash]*schedulingTypes.Message, sequences []*types.ExecutingSequence, timestamp *big.Int, msgTemplate *actor.Message, inlog *actor.WorkerThreadLogger, parallelism int, generationIdx, batchIdx int) (map[ethCommon.Hash]*types.ExecuteResponse, map[ethCommon.Hash]ethCommon.Hash, map[ethCommon.Hash][]ethCommon.Hash, []ethCommon.Address, map[ethCommon.Hash]uint32, map[ethCommon.Hash]ethCommon.Address)
}
