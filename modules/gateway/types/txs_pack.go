package types

import (
	cmntyp "github.com/arcology-network/common-lib/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type TxsPack struct {
	Txs        *cmntyp.IncomingTxs
	TxHashChan chan evmCommon.Hash
}
