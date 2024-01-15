package tools

import (
	"github.com/arcology-network/common-lib/exp/array"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func CalculateHash(hashes []*evmCommon.Hash) evmCommon.Hash {
	return evmCommon.BytesToHash(crypto.Keccak256(array.Concate(hashes, func(v *evmCommon.Hash) []byte { return (*v)[:] })))
}
