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

package backend

import (
	"math/big"

	mtypes "github.com/arcology-network/main/types"
	ccdb "github.com/arcology-network/storage-committer/storage/ethstorage"
	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/engine"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtyp "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/params"
)

type EthereumAPI interface {
	BlockNumber() (uint64, error)
	// If fullTx is true, the actual type of RPCBlock.Transactions is []*RPCTransaction.
	// If fullTx is false, the actual type of RPCBlock.Transactions is []ethcmn.Hash.
	GetBlockByNumber(number int64, fullTx bool) (*mtypes.RPCBlock, error)
	GetBlockByHash(hash ethcmn.Hash, fullTx bool) (*mtypes.RPCBlock, error)
	GetHeaderByNumber(number int64) (*mtypes.RPCBlock, error)
	GetHeaderByHash(hash ethcmn.Hash) (*mtypes.RPCBlock, error)
	GetCode(address ethcmn.Address, number int64) ([]byte, error)
	GetBalance(address ethcmn.Address, number int64) (*big.Int, error)
	GetTransactionCount(address ethcmn.Address, number int64) (uint64, error)
	GetStorageAt(address ethcmn.Address, key string, number int64) ([]byte, error)

	EstimateGas(msg eth.CallMsg) (uint64, error)
	GasPrice() (*big.Int, error)

	GetTransactionByHash(hash ethcmn.Hash) (*mtypes.RPCTransaction, error)

	Call(msg eth.CallMsg) ([]byte, error)
	SendRawTransaction(rawTx []byte) (ethcmn.Hash, error)
	GetTransactionReceipt(hash ethcmn.Hash) (*ethtyp.Receipt, error)
	GetBlockReceipts(height uint64) ([]*ethtyp.Receipt, error)
	GetLogs(filter eth.FilterQuery) ([]*ethtyp.Log, error)

	GetBlockTransactionCountByHash(hash ethcmn.Hash) (int, error)
	GetBlockTransactionCountByNumber(number int64) (int, error)

	GetTransactionByBlockHashAndIndex(hash ethcmn.Hash, index int) (*mtypes.RPCTransaction, error)
	GetTransactionByBlockNumberAndIndex(number int64, index int) (*mtypes.RPCTransaction, error)

	GetUncleCountByBlockHash(hash ethcmn.Hash) (int, error)
	GetUncleCountByBlockNumber(number int64) (int, error)

	SubmitWork() (bool, error)
	SubmitHashrate() (bool, error)

	Hashrate() (int, error)
	GetWork() ([]string, error)
	ProtocolVersion() (int, error)
	//Coinbase() (string, error)

	Syncing() (bool, error)
	Proposer() (bool, error)

	NewFilter(filter eth.FilterQuery) (ID, error)
	NewBlockFilter() (ID, error)
	NewPendingTransactionFilter() (ID, error)
	UninstallFilter(id ID) (bool, error)
	GetFilterChanges(id ID) (interface{}, error)
	GetFilterLogs(id ID) ([]*ethtyp.Log, error)

	ForkchoiceUpdatedV2(update engine.ForkchoiceStateV1, payloadAttributes *engine.PayloadAttributes, chainid uint64) (engine.ForkChoiceResponse, error)
	GetPayloadV2(payloadID engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error)
	NewPayloadV2(params engine.ExecutableData) (engine.PayloadStatusV1, error)
	SignalSuperchainV1(signal *catalyst.SuperchainSignal) (params.ProtocolVersion, error)

	GetProof(rq *mtypes.RequestProof) (*ccdb.AccountResult, error)

	SendRawTransactions(rawTxs [][]byte) (uint64, error)
}
