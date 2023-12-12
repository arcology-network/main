package scheduler

import (
	"math/big"

	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	schedulingTypes "github.com/arcology-network/main/modules/scheduler/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type Arbitrator interface {
	Start()
	Stop()
	Do(els [][][]*types.TxElement, log *actor.WorkerThreadLogger, generationIdx int, batchIdx int) ([]*evmCommon.Hash, []uint32, []uint32)
}

type Executor interface {
	Start()
	Stop()
	// Run(msgs map[evmCommon.Hash]*schedulingTypes.Message, sequences []*types.ExecutingSequence, timestamp *big.Int, msgTemplate *actor.Message, inlog *actor.WorkerThreadLogger, parallelism int, generationIdx, batchIdx int) (map[evmCommon.Hash]*types.ExecuteResponse, map[evmCommon.Hash]evmCommon.Hash, map[evmCommon.Hash][]evmCommon.Hash, []evmCommon.Address, map[evmCommon.Hash]uint32, map[evmCommon.Hash]evmCommon.Address)
	Run(msgs map[evmCommon.Hash]*schedulingTypes.Message, sequences []*types.ExecutingSequence, timestamp *big.Int, msgTemplate *actor.Message, inlog *actor.WorkerThreadLogger, parallelism int, generationIdx, batchIdx int) (map[evmCommon.Hash]*types.ExecuteResponse, []evmCommon.Address)
}
