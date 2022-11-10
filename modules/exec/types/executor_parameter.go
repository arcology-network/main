package types

import (
	"github.com/arcology-network/common-lib/types"
	mevmCommon "github.com/arcology-network/evm/common"
)

type ExecutorParameter struct {
	ParentInfo *types.ParentInfo
	Coinbase   *mevmCommon.Address
	Height     uint64
}
