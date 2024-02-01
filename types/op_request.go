package types

import (
	"github.com/arcology-network/common-lib/types"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type OpRequest struct {
	BlockParam   *BlockParams
	Withdrawals  evmTypes.Withdrawals // List of withdrawals to include in block.
	Transactions []*types.StandardTransaction
}
