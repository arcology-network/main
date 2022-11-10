package types

import (
	"math/big"

	ethCommon "github.com/arcology-network/3rd-party/eth/common"
	"github.com/arcology-network/common-lib/types"
	urlcommon "github.com/arcology-network/concurrenturl/v2/common"
)

type TxToExecutes struct {
	Sequences     []*types.ExecutingSequence
	SnapShots     *urlcommon.DatastoreInterface
	PrecedingHash ethCommon.Hash
	Timestamp     *big.Int
	Parallelism   uint64
	Debug         bool
}
