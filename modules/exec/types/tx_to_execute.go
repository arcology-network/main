package types

import (
	"math/big"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/types"
	urlcommon "github.com/HPISTechnologies/concurrenturl/v2/common"
)

type TxToExecutes struct {
	Sequences     []*types.ExecutingSequence
	SnapShots     *urlcommon.DatastoreInterface
	PrecedingHash ethCommon.Hash
	Timestamp     *big.Int
	Parallelism   uint64
	Debug         bool
}
