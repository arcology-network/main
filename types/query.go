package types

import (
	"math/big"
	"time"

	"github.com/arcology-network/common-lib/types"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

const (
	QueryType_Receipt      = "receipt"
	QueryType_Balance      = "balance"
	QueryType_Container    = "container"
	QueryType_Block        = "block"
	QueryType_RawBlock     = "rawblock"
	QueryType_Nonce        = "nonce"
	QueryType_LatestHeight = "latestheight"

	QueryType_BlockNumber      = "blocknumber"
	QueryType_Code             = "code"
	QueryType_Balance_Eth      = "balanceEth"
	QueryType_TransactionCount = "transactionCount"
	QueryType_Storage          = "storage"
	QueryType_Receipt_Eth      = "receiptEth"
	QueryType_Block_Receipts   = "blockReceipts"
	QueryType_Transaction      = "transaction"
	QueryType_Block_Eth        = "blockEth"
	QueryType_HeaderByHash     = "headerByHash"
	QueryType_HeaderByNumber   = "headerByNumber"
	QueryType_BlocByHash       = "blockbyhash"
	QueryType_Logs             = "logs"

	QueryType_TxNumsByHash     = "txNumsByHash"
	QueryType_TxNumsByNumber   = "txNumsByNumber"
	QueryType_TxByHashAndIdx   = "txByHashAndIdx"
	QueryType_TxByNumberAndIdx = "txByNumberAndIdx"

	QueryType_Proof = "proof"

	ConcurrentLibStyle_Array = "array"
	ConcurrentLibStyle_Map   = "map"
	ConcurrentLibStyle_Queue = "queue"

	QueryType_Syncing  = "syncing"
	QueryType_Proposer = "proposer"
)

type QueryRequest struct {
	QueryType string
	Data      interface{}
}

type QueryResult struct {
	Data interface{}
}

type RequestParameters struct {
	Number  int64
	Address ethCommon.Address
}

type RequestBlockEth struct {
	Number int64
	Hash   ethCommon.Hash
	Index  int
	FullTx bool
}
type RequestStorage struct {
	Number  int64
	Address ethCommon.Address
	Key     string
}

type RequestBalance struct {
	Height  int
	Address string
}

type RequestContainer struct {
	Height  int
	Address string
	Id      string
	Style   string
	Key     string
}

type RequestBlock struct {
	Height       int
	Transactions bool
}

type RequestReceipt struct {
	Hashes             []string
	ExecutingDebugLogs bool
}

// type RequestProof struct {
// 	Address  ethCommon.Address `json:"address"`
// 	Keys     []ethCommon.Hash  `json:"storageKeys"`
// 	BlockTag ethCommon.Hash    `json:"blockParameter"`
// }

type Block struct {
	Height       int           `json:"height"`
	Hash         string        `json:"hash"`
	Coinbase     string        `json:"coinbase"`
	Number       int           `json:"number"`
	Transactions []string      `json:"transactions"`
	GasUsed      *big.Int      `json:"gasUsed"`
	ExecTime     time.Duration `json:"elapsedTime"`
	Timestamp    int           `json:"timestamp"`
}

type Log struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	BlockNumber uint64   `json:"blockNumber"`
	TxHash      string   `json:"transactionHash"`
	TxIndex     uint     `json:"transactionIndex"`
	BlockHash   string   `json:"blockHash"`
	Index       uint     `json:"logIndex"`
}

type QueryReceipt struct {
	Status          int      `json:"status"`
	ContractAddress string   `json:"contractAddress"`
	GasUsed         *big.Int `json:"gasUsed"`
	Logs            []*Log   `json:"logs"`
	ExecutingLogs   string   `json:"executing logs"`
	Height          int      `json:"height"`
}

type ClusterConfig struct {
	Parallelism int
}

type SetReply struct {
	Status int
}

type SendTransactionArgs struct {
	Txs [][]byte
}

type SendTransactionReply struct {
	Status int
	Data   interface{}
}

type RawTransactionArgs struct {
	Tx  []byte
	Src types.TxSource
}
type RawTransactionReply struct {
	TxHash interface{}
}
