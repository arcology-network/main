package types

import (
	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
)

type TxsPack struct {
	Txs        [][]byte
	TxHashChan chan ethCommon.Hash
}
