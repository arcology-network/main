/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package types

import (
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtyp "github.com/ethereum/go-ethereum/core/types"
)

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash           *ethcmn.Hash       `json:"blockHash"`
	BlockNumber         *hexutil.Big       `json:"blockNumber"`
	From                ethcmn.Address     `json:"from"`
	Gas                 hexutil.Uint64     `json:"gas"`
	GasPrice            *hexutil.Big       `json:"gasPrice"`
	GasFeeCap           *hexutil.Big       `json:"maxFeePerGas,omitempty"`
	GasTipCap           *hexutil.Big       `json:"maxPriorityFeePerGas,omitempty"`
	MaxFeePerBlobGas    *hexutil.Big       `json:"maxFeePerBlobGas,omitempty"`
	Hash                ethcmn.Hash        `json:"hash"`
	Input               hexutil.Bytes      `json:"input"`
	Nonce               hexutil.Uint64     `json:"nonce"`
	To                  *ethcmn.Address    `json:"to"`
	TransactionIndex    *hexutil.Uint64    `json:"transactionIndex"`
	Value               *hexutil.Big       `json:"value"`
	Type                hexutil.Uint64     `json:"type"`
	Accesses            *ethtyp.AccessList `json:"accessList,omitempty"`
	ChainID             *hexutil.Big       `json:"chainId,omitempty"`
	BlobVersionedHashes []ethcmn.Hash      `json:"blobVersionedHashes,omitempty"`
	V                   *hexutil.Big       `json:"v"`
	R                   *hexutil.Big       `json:"r"`
	S                   *hexutil.Big       `json:"s"`
	YParity             *hexutil.Uint64    `json:"yParity,omitempty"`

	// deposit-tx only
	SourceHash *ethcmn.Hash `json:"sourceHash,omitempty"`
	Mint       *hexutil.Big `json:"mint,omitempty"`
	IsSystemTx *bool        `json:"isSystemTx,omitempty"`
	// deposit-tx post-Canyon only
	DepositReceiptVersion *hexutil.Uint64 `json:"depositReceiptVersion,omitempty"`
}

type RPCBlock struct {
	Header       *ethtyp.Header
	Transactions []interface{}
}

type FeeHistoryResult struct {
	OldestBlock  *hexutil.Big     `json:"oldestBlock"`
	Reward       [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee      []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64        `json:"gasUsedRatio"`
}

const (
	BlockNumberLatest    = -1
	BlockNumberEarliest  = -2
	BlockNumberPending   = -3
	BlockNumberFinalized = -4
	BlockNumberSafe      = -5
)
