package types

import (
	"math/big"

	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

const (
	Signer_London = iota
	Signer_Cancun
)

func DefaultSignerType() uint8 {
	return Signer_London
}

func MakeSigner(signerType uint8, chainId *big.Int) evmTypes.Signer {
	switch signerType {
	case Signer_London:
		return evmTypes.NewLondonSigner(chainId)
	case Signer_Cancun:
		return evmTypes.NewCancunSigner(chainId)
	}
	return nil
}

func GetSignerType(height *big.Int, config *params.ChainConfig) uint8 {
	var signerType uint8

	switch {
	case isCancun():
		signerType = Signer_Cancun
	default:
		signerType = Signer_London
	}
	return signerType
}

func isCancun() bool {
	return false
}
