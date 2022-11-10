package types

import (
	ethCommon "github.com/arcology-network/3rd-party/eth/common"
	cmntyp "github.com/arcology-network/common-lib/types"
)

type TxsPack struct {
	Txs        *cmntyp.IncomingTxs
	TxHashChan chan ethCommon.Hash
}
