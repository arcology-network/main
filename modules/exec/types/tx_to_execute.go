package types

import (
	"math/big"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/storage-committer/interfaces"
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
