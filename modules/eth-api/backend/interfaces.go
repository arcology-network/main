package backend

import (
	"math/big"

	"github.com/arcology-network/component-lib/ethrpc"
	eth "github.com/ethereum/go-ethereum"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtyp "github.com/ethereum/go-ethereum/core/types"
)

type EthereumAPI interface {
	BlockNumber() (uint64, error)
	// If fullTx is true, the actual type of RPCBlock.Transactions is []*RPCTransaction.
	// If fullTx is false, the actual type of RPCBlock.Transactions is []ethcmn.Hash.
	GetBlockByNumber(number int64, fullTx bool) (*ethrpc.RPCBlock, error)
	GetBlockByHash(hash ethcmn.Hash, fullTx bool) (*ethrpc.RPCBlock, error)
	GetHeaderByNumber(number int64) (*ethrpc.RPCBlock, error)
	GetHeaderByHash(hash ethcmn.Hash) (*ethrpc.RPCBlock, error)
	GetCode(address ethcmn.Address, number int64) ([]byte, error)
	GetBalance(address ethcmn.Address, number int64) (*big.Int, error)
	GetTransactionCount(address ethcmn.Address, number int64) (uint64, error)
	GetStorageAt(address ethcmn.Address, key string, number int64) ([]byte, error)

	EstimateGas(msg eth.CallMsg) (uint64, error)
	GasPrice() (*big.Int, error)

	GetTransactionByHash(hash ethcmn.Hash) (*ethrpc.RPCTransaction, error)

	Call(msg eth.CallMsg) ([]byte, error)
	SendRawTransaction(rawTx []byte) (ethcmn.Hash, error)
	GetTransactionReceipt(hash ethcmn.Hash) (*ethtyp.Receipt, error)
	GetBlockReceipts(height uint64) ([]*ethtyp.Receipt, error)
	GetLogs(filter eth.FilterQuery) ([]*ethtyp.Log, error)

	GetBlockTransactionCountByHash(hash ethcmn.Hash) (int, error)
	GetBlockTransactionCountByNumber(number int64) (int, error)

	GetTransactionByBlockHashAndIndex(hash ethcmn.Hash, index int) (*ethrpc.RPCTransaction, error)
	GetTransactionByBlockNumberAndIndex(number int64, index int) (*ethrpc.RPCTransaction, error)

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
}
