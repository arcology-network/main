package backend

import (
	"math/big"
	"time"

	cmncmn "github.com/arcology-network/common-lib/common"
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/ethrpc"
	intf "github.com/arcology-network/component-lib/interface"
	eth "github.com/ethereum/go-ethereum"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethcrp "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/core/types"
)

type Monaco struct {
	filters *Filters
}

func NewMonaco(filters *Filters) *Monaco {
	return &Monaco{
		filters: filters,
	}
}

func (m *Monaco) BlockNumber() (uint64, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_BlockNumber,
	}, &response)
	if err != nil {
		return 0, err
	}
	return response.Data.(uint64), nil
}

func (m *Monaco) GetBlockByNumber(number int64, fullTx bool) (*ethrpc.RPCBlock, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_Block_Eth,
		Data: &cmntyp.RequestBlockEth{
			Number: number,
			FullTx: fullTx,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*ethrpc.RPCBlock), nil
}

func (m *Monaco) GetBlockByHash(hash ethcmn.Hash, fullTx bool) (*ethrpc.RPCBlock, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_BlocByHash,
		Data: &cmntyp.RequestBlockEth{
			Hash:   hash,
			FullTx: fullTx,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*ethrpc.RPCBlock), nil
}

func (m *Monaco) GetHeaderByNumber(number int64) (*ethrpc.RPCBlock, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_HeaderByNumber,
		Data: &cmntyp.RequestBlockEth{
			Number: number,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*ethrpc.RPCBlock), nil
}

func (m *Monaco) GetHeaderByHash(hash ethcmn.Hash) (*ethrpc.RPCBlock, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_HeaderByHash,
		Data: &cmntyp.RequestBlockEth{
			Hash: hash,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*ethrpc.RPCBlock), nil
}

func (m *Monaco) GetCode(address ethcmn.Address, number int64) ([]byte, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_Code,
		Data: cmntyp.RequestParameters{
			Number:  number,
			Address: address,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.([]byte), nil
}

func (m *Monaco) GetBalance(address ethcmn.Address, number int64) (*big.Int, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_Balance_Eth,
		Data: &cmntyp.RequestParameters{
			Number:  number,
			Address: address,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*big.Int), nil
}

func (m *Monaco) GetTransactionCount(address ethcmn.Address, number int64) (uint64, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_TransactionCount,
		Data: cmntyp.RequestParameters{
			Number:  number,
			Address: address,
		},
	}, &response)
	if err != nil {
		return 0, err
	}
	return response.Data.(uint64), nil
}

func (m *Monaco) GetStorageAt(address ethcmn.Address, key string, number int64) ([]byte, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_Storage,
		Data: cmntyp.RequestStorage{
			Number:  number,
			Address: address,
			Key:     key,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.([]byte), nil
}

func (m *Monaco) EstimateGas(msg eth.CallMsg) (uint64, error) {
	// TODO
	return 0x10000000, nil
}

func (m *Monaco) GasPrice() (*big.Int, error) {
	// TODO
	return new(big.Int).SetUint64(0xff), nil
}

func (m *Monaco) GetTransactionByHash(hash ethcmn.Hash) (*ethrpc.RPCTransaction, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_Transaction,
		Data:      hash,
	}, &response)
	if err != nil {
		err = intf.Router.Call("pool", "Query", &cmntyp.QueryRequest{
			QueryType: cmntyp.QueryType_Transaction,
			Data:      hash,
		}, &response)
		if err != nil {
			return nil, err
		}
	}
	return response.Data.(*ethrpc.RPCTransaction), nil
}

func (m *Monaco) Call(msg eth.CallMsg) ([]byte, error) {
	var response cmntyp.ExecutorResponses
	var to *ethcmn.Address
	if msg.To != nil {
		addr := ethcmn.BytesToAddress(msg.To.Bytes())
		to = &addr
	}
	message := core.NewMessage(
		ethcmn.BytesToAddress(msg.From.Bytes()),
		to,
		1,
		msg.Value,
		msg.Gas,
		msg.GasPrice,
		msg.Data,
		nil,
		false,
	)
	hash, _ := msgHash(&message)
	err := intf.Router.Call("executor-1", "ExecTxs", &actor.Message{
		Height: 0,
		Name:   actor.MsgTxsToExecute,
		Msgid:  cmncmn.GenerateUUID(),
		Data: &cmntyp.ExecutorRequest{
			Sequences: []*cmntyp.ExecutingSequence{
				{
					Msgs: []*cmntyp.StandardMessage{
						{
							TxHash: hash,
							Native: &message,
						},
					},
					Parallel:   true,
					SequenceId: hash,
					Txids:      []uint32{0},
				},
			},
			Precedings:    [][]*ethcmn.Hash{nil},
			PrecedingHash: []ethcmn.Hash{{}},
			Timestamp:     new(big.Int).SetInt64(time.Now().Unix()),
			Parallelism:   1,
			Debug:         true,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.CallResults[0], nil
}

func (m *Monaco) SendRawTransaction(rawTx []byte) (ethcmn.Hash, error) {
	var response cmntyp.RawTransactionReply
	err := intf.Router.Call("gateway", "SendRawTransaction", &cmntyp.RawTransactionArgs{
		Tx: rawTx,
	}, &response)
	if err != nil {
		return ethcmn.Hash{}, err
	}
	return response.TxHash.(ethcmn.Hash), nil
}

func (m *Monaco) GetTransactionReceipt(hash ethcmn.Hash) (*types.Receipt, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_Receipt_Eth,
		Data:      hash,
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*types.Receipt), nil
}

func (m *Monaco) GetBlockReceipts(height uint64) ([]*types.Receipt, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_Block_Receipts,
		Data:      height,
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.([]*types.Receipt), nil
}

func (m *Monaco) GetLogs(filter eth.FilterQuery) ([]*types.Log, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_Logs,
		Data:      &filter,
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.([]*types.Log), nil
}

func (m *Monaco) GetTransactionByBlockHashAndIndex(hash ethcmn.Hash, index int) (*ethrpc.RPCTransaction, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_TxByHashAndIdx,
		Data: &cmntyp.RequestBlockEth{
			Hash:  hash,
			Index: index,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*ethrpc.RPCTransaction), nil
}
func (m *Monaco) GetTransactionByBlockNumberAndIndex(number int64, index int) (*ethrpc.RPCTransaction, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_TxByNumberAndIdx,
		Data: &cmntyp.RequestBlockEth{
			Number: number,
			Index:  index,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*ethrpc.RPCTransaction), nil
}

func (m *Monaco) GetBlockTransactionCountByHash(hash ethcmn.Hash) (int, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_TxNumsByHash,
		Data:      hash,
	}, &response)
	if err != nil {
		return 0, err
	}
	return response.Data.(int), nil
}
func (m *Monaco) GetBlockTransactionCountByNumber(number int64) (int, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("storage", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_TxNumsByNumber,
		Data:      number,
	}, &response)
	if err != nil {
		return 0, err
	}
	return response.Data.(int), nil
}

type messageRLP struct {
	To         *ethcmn.Address
	From       ethcmn.Address
	Nonce      uint64
	Amount     *big.Int
	GasLimit   uint64
	GasPrice   *big.Int
	Data       []byte
	CheckNonce bool
}

func msgHash(msg *core.Message) (ethcmn.Hash, error) {
	var hash ethcmn.Hash
	sha := sha3.NewLegacyKeccak256().(ethcrp.KeccakState)
	rlp.Encode(sha, &messageRLP{
		To:         msg.To,
		From:       msg.From,
		Nonce:      msg.Nonce,
		Amount:     msg.Value,
		GasLimit:   msg.GasLimit,
		GasPrice:   msg.GasPrice,
		Data:       msg.Data,
		CheckNonce: !msg.SkipAccountChecks,
	})
	sha.Read(hash[:])
	return hash, nil
}

func (m *Monaco) GetUncleCountByBlockHash(hash ethcmn.Hash) (int, error) {
	return 0x0, nil
}
func (m *Monaco) GetUncleCountByBlockNumber(number int64) (int, error) {
	return 0x0, nil
}
func (m *Monaco) SubmitWork() (bool, error) {
	return true, nil
}
func (m *Monaco) SubmitHashrate() (bool, error) {
	return true, nil
}
func (m *Monaco) Hashrate() (int, error) {
	return 0x3e6, nil
}
func (m *Monaco) GetWork() ([]string, error) {
	return []string{}, nil
}
func (m *Monaco) ProtocolVersion() (int, error) {
	return 10000 + 2, nil
}

//	func (m *Monaco) Coinbase() (string, error) {
//		return 10000 + 2, nil
//	}
func (m *Monaco) Syncing() (bool, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("consensus", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_Syncing,
	}, &response)
	if err != nil {
		return false, err
	}
	return response.Data.(bool), nil
}

func (m *Monaco) Proposer() (bool, error) {
	var response cmntyp.QueryResult
	err := intf.Router.Call("consensus", "Query", &cmntyp.QueryRequest{
		QueryType: cmntyp.QueryType_Proposer,
	}, &response)
	if err != nil {
		return false, err
	}
	return response.Data.(bool), nil
}

func (m *Monaco) NewFilter(filter eth.FilterQuery) (ID, error) {
	return m.filters.NewFilter(filter), nil
}
func (m *Monaco) NewBlockFilter() (ID, error) {
	return m.filters.NewBlockFilter(), nil
}
func (m *Monaco) NewPendingTransactionFilter() (ID, error) {
	return m.filters.NewPendingTransactionFilter(), nil
}
func (m *Monaco) UninstallFilter(id ID) (bool, error) {
	return m.filters.UninstallFilter(id), nil
}
func (m *Monaco) GetFilterChanges(id ID) (interface{}, error) {
	return m.filters.GetFilterChanges(id)
}
func (m *Monaco) GetFilterLogs(id ID) ([]*types.Log, error) {
	crit, err := m.filters.GetFilterLogsCrit(id)
	if err != nil {
		return nil, err
	}
	logs, err := m.GetLogs(*crit)
	if err != nil {
		return nil, err
	}
	return returnLogs(logs), nil
}
