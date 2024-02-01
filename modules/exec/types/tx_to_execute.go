package types

import (
	"math/big"

	"github.com/arcology-network/concurrenturl/interfaces"
	mtypes "github.com/arcology-network/main/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type TxToExecutes struct {
	Sequences     []*mtypes.ExecutingSequence
	SnapShots     *interfaces.Datastore
	PrecedingHash evmCommon.Hash
	Timestamp     *big.Int
	Parallelism   uint64
	Debug         bool
}
