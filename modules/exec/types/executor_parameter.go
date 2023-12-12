package types

import (
	"github.com/arcology-network/common-lib/types"
	mevmCommon "github.com/ethereum/go-ethereum/common"
)

type ExecutorParameter struct {
	ParentInfo *types.ParentInfo
	Coinbase   *mevmCommon.Address
	Height     uint64
}
