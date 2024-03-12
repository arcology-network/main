package tools

import (
	"github.com/arcology-network/common-lib/exp/slice"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func CalculateHash(hashes []*evmCommon.Hash) evmCommon.Hash {
	if len(hashes) == 0 {
		return evmCommon.Hash{}
	}
	return evmCommon.BytesToHash(crypto.Keccak256(slice.Concate(hashes, func(v *evmCommon.Hash) []byte { return (*v)[:] })))
}
