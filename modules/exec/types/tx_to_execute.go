package types

import (
	"math/big"

	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/concurrenturl/interfaces"
	evmCommon "github.com/arcology-network/evm/common"
)

type TxToExecutes struct {
	Sequences     []*types.ExecutingSequence
	SnapShots     *interfaces.Datastore
	PrecedingHash evmCommon.Hash
	Timestamp     *big.Int
	Parallelism   uint64
	Debug         bool
}
