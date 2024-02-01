package types

import (
	"math/big"

	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type BlockResult struct {
	Block *evmTypes.Block
	Fees  *big.Int // total block fees
}
