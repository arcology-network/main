package scheduler

import (
	"math/big"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type Arbitrator interface {
	Start()
	Stop()
	Do(els [][]evmCommon.Hash, log *actor.WorkerThreadLogger, generationIdx int) ([]evmCommon.Hash, []uint32, []uint32)
}

type Executor interface {
	Start()
	Stop()
	Run(sequences []*mtypes.ExecutingSequence, timestamp *big.Int, msgTemplate *actor.Message, inlog *actor.WorkerThreadLogger, parallelism int, height uint64, generationIdx int) (map[evmCommon.Hash]*mtypes.ExecuteResponse, []evmCommon.Address)
}
