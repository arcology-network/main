package types

import (
	mtypes "github.com/arcology-network/main/types"
	mevmCommon "github.com/ethereum/go-ethereum/common"
)

type ExecutorParameter struct {
	ParentInfo *mtypes.ParentInfo
	Coinbase   *mevmCommon.Address
	Height     uint64
}
