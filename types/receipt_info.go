package types

import (
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type ReceiptInfo struct {
	RcptHash  evmCommon.Hash
	BloomInfo evmTypes.Bloom
	Gasused   uint64
	TpsGas    *TPSGasBurned
}
