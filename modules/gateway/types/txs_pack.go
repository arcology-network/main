package types

import (
	cmntyp "github.com/arcology-network/common-lib/types"
	evmCommon "github.com/arcology-network/evm/common"
)

type TxsPack struct {
	Txs        *cmntyp.IncomingTxs
	TxHashChan chan evmCommon.Hash
}
