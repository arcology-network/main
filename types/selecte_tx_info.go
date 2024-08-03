package types

import (
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type SelectedTxsInfo struct {
	Txhash evmCommon.Hash
	Txs    [][]byte
}
