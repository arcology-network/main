package types

import (
	ethCommon "github.com/ethereum/go-ethereum/common"
)

type ParentInfo struct {
	ParentRoot ethCommon.Hash
	ParentHash ethCommon.Hash
}
