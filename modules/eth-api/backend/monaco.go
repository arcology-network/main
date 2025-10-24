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
	"fmt"
	"math/big"
	"sync"
	"time"

	mtypes "github.com/arcology-network/main/types"
	ccdb "github.com/arcology-network/storage-committer/storage/ethstorage"
	intf "github.com/arcology-network/streamer/interface"
	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/engine"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethcrp "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"

	eucommon "github.com/arcology-network/common-lib/types"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/catalyst"
)

const (
	TxGas        = 21000
	zoomout      = 2
	defaultPrice = 255
)

type Monaco struct {
	filters     *Filters
	localBlocks *payloadQueue // Cache of local payloads generated

	forkchoiceLock sync.Mutex // Lock for the forkChoiceUpdated method
	newPayloadLock sync.Mutex // Lock for the NewPayload method
}

func NewMonaco(filters *Filters) *Monaco {
	return &Monaco{
		filters:     filters,
		localBlocks: newPayloadQueue(),
	}
}

func (api *Monaco) GetProof(rq *mtypes.RequestProof) (*ccdb.AccountResult, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("state_query", "QueryState", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Proof,
		Data:      rq,
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*ccdb.AccountResult), nil
}

// ForkchoiceUpdatedV2 is equivalent to V1 with the addition of withdrawals in the payload attributes.
func (api *Monaco) ForkchoiceUpdatedV2(update engine.ForkchoiceStateV1, payloadAttributes *engine.PayloadAttributes, chainid uint64) (engine.ForkChoiceResponse, error) {
	api.forkchoiceLock.Lock()
	defer api.forkchoiceLock.Unlock()
	valid := func(id *engine.PayloadID) engine.ForkChoiceResponse {
		return engine.ForkChoiceResponse{
			PayloadStatus: engine.PayloadStatusV1{Status: engine.VALID, LatestValidHash: &update.HeadBlockHash},
			PayloadID:     id,
		}
	}
	// If payload generation was requested, create a new block to be potentially
	// sealed by the beacon client. The payload will be requested later, and we
	// will replace it arbitrarily many times in between.
	if payloadAttributes != nil {

		transactions := make(types.Transactions, 0, len(payloadAttributes.Transactions))
		for i, otx := range payloadAttributes.Transactions {
			var tx types.Transaction
			if err := tx.UnmarshalBinary(otx); err != nil {
				return engine.STATUS_INVALID, fmt.Errorf("transaction %d is not valid: %v", i, err)
			}
			transactions = append(transactions, &tx)
		}
		args := &miner.BuildPayloadArgs{
			Parent:       update.HeadBlockHash,
			Timestamp:    payloadAttributes.Timestamp,
			FeeRecipient: payloadAttributes.SuggestedFeeRecipient,
			Random:       payloadAttributes.Random,
			Withdrawals:  payloadAttributes.Withdrawals,
			BeaconRoot:   payloadAttributes.BeaconRoot,
			NoTxPool:     payloadAttributes.NoTxPool,
			Transactions: transactions,
			GasLimit:     payloadAttributes.GasLimit,
		}
		id := args.Id()
		// If we already are busy generating this work, then we do not need
		// to start a second process.
		if api.localBlocks.has(id) {
			return valid(&id), nil
		}
		payload, err := buildPayload(args, payloadAttributes.Transactions, chainid)
		if err != nil {
			log.Error("Failed to build payload", "err", err)
			return valid(nil), engine.InvalidPayloadAttributes.With(err)
		}

		api.localBlocks.put(id, payload)
		return valid(&id), nil
	}
	return valid(nil), nil
}

// GetPayloadV2 returns a cached payload by id.
func (api *Monaco) GetPayloadV2(payloadID engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	log.Trace("Engine API request received", "method", "GetPayload", "id", payloadID)
	data := api.localBlocks.get(payloadID, false)
	if data == nil {
		return nil, engine.UnknownPayload
	}
	return data, nil
}

// invalid returns a response "INVALID" with the latest valid hash supplied by latest.
func (api *Monaco) invalid(err error, latestValid *types.Header) engine.PayloadStatusV1 {
	var currentHash *ethcmn.Hash
	if latestValid != nil {
		if latestValid.Difficulty.BitLen() != 0 {
			// Set latest valid hash to 0x0 if parent is PoW block
			currentHash = &ethcmn.Hash{}
		} else {
			// Otherwise set latest valid hash to parent hash
			h := latestValid.Hash()
			currentHash = &h
		}
	}
	errorMsg := err.Error()
	return engine.PayloadStatusV1{Status: engine.INVALID, LatestValidHash: currentHash, ValidationError: &errorMsg}
}

// NewPayloadV2 creates an Eth1 block, inserts it in the chain, and returns the status of the chain.
func (api *Monaco) NewPayloadV2(params engine.ExecutableData) (engine.PayloadStatusV1, error) {
	// api.newPayloadLock.Lock()
	// defer api.newPayloadLock.Unlock()

	// log.Trace("Engine API request received", "method", "NewPayload", "number", params.Number, "hash", params.BlockHash)
	// block, err := engine.ExecutableDataToBlock(params, nil, nil)
	// if err != nil {
	// 	log.Warn("Invalid NewPayload params", "params", params, "error", err)
	// 	return api.invalid(err, nil), nil
	// }
	// hash := block.Hash()

	counter := 20
	var latestHeight uint64
	for {
		var response mtypes.QueryResult
		err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
			QueryType: mtypes.QueryType_LatestHeight,
		}, &response)
		if err != nil {
			time.Sleep(200 * time.Millisecond)
		}

		latestHeight = response.Data.(uint64)
		// fmt.Printf("----------main/modules/eth-api/backend/monaco.go----NewPayloadV2---------latestHeight:%v,params.Number:%v\n", latestHeight, params.Number)
		if latestHeight >= params.Number {
			break
		}
		counter--
		if counter <= 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	return engine.PayloadStatusV1{Status: engine.VALID, LatestValidHash: &params.BlockHash}, nil
}

func (api *Monaco) SignalSuperchainV1(signal *catalyst.SuperchainSignal) (params.ProtocolVersion, error) {
	return params.OPStackSupport, nil
}

func (m *Monaco) BlockNumber() (uint64, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_BlockNumber,
	}, &response)
	if err != nil {
		return 0, err
	}
	return response.Data.(uint64), nil
}

func (m *Monaco) GetBlockByNumber(number int64, fullTx bool) (*mtypes.RPCBlock, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Block_Eth,
		Data: &mtypes.RequestBlockEth{
			Number: number,
			FullTx: fullTx,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*mtypes.RPCBlock), nil
}

func (m *Monaco) GetBlockByHash(hash ethcmn.Hash, fullTx bool) (*mtypes.RPCBlock, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_BlocByHash,
		Data: &mtypes.RequestBlockEth{
			Hash:   hash,
			FullTx: fullTx,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*mtypes.RPCBlock), nil
}

func (m *Monaco) GetHeaderByNumber(number int64) (*mtypes.RPCBlock, error) {
	return GetHeaderByNumber(number)
}

func GetHeaderByNumber(number int64) (*mtypes.RPCBlock, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_HeaderByNumber,
		Data: &mtypes.RequestBlockEth{
			Number: number,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*mtypes.RPCBlock), nil
}

func GetHeaderFromHash(hash ethcmn.Hash) (*mtypes.RPCBlock, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_HeaderByHash,
		Data: &mtypes.RequestBlockEth{
			Hash: hash,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*mtypes.RPCBlock), nil
}

func (m *Monaco) GetHeaderByHash(hash ethcmn.Hash) (*mtypes.RPCBlock, error) {
	return GetHeaderFromHash(hash)
}

func (m *Monaco) GetCode(address ethcmn.Address, number int64) ([]byte, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Code,
		Data: mtypes.RequestParameters{
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
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Balance_Eth,
		Data: &mtypes.RequestParameters{
			Number:  number,
			Address: address,
		},
	}, &response)
	if err != nil {
		return big.NewInt(0), nil
	}
	return response.Data.(*big.Int), nil
}

func (m *Monaco) GetTransactionCount(address ethcmn.Address, number int64) (uint64, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TransactionCount,
		Data: mtypes.RequestParameters{
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
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Storage,
		Data: mtypes.RequestStorage{
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
	// return 0x10000000, nil

	// If the transaction is a plain value transfer, short circuit estimation and directly try 21000.
	if len(msg.Data) == 0 {
		return TxGas, nil
	}

	//set GasPrice and GasLimit
	request, maxGasLimit := m.callmsgToRequest(msg)

	// try run
	var response core.ExecutionResult
	err := intf.Router.Call("estimate-executor", "ExecTxs", request, &response)
	if err != nil {
		return uint64(0), err
	}
	gas := response.UsedGas * zoomout
	if gas > maxGasLimit {
		gas = maxGasLimit
	}
	return gas, response.Err

}

func (m *Monaco) GasPrice() (*big.Int, error) {
	// TODO
	return new(big.Int).SetUint64(defaultPrice), nil
}

func (m *Monaco) GetTransactionByHash(hash ethcmn.Hash) (*mtypes.RPCTransaction, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Transaction,
		Data:      hash,
	}, &response)
	if err != nil {
		err = intf.Router.Call("pool", "Query", &mtypes.QueryRequest{
			QueryType: mtypes.QueryType_Transaction,
			Data:      hash,
		}, &response)
		if err != nil {
			return nil, err
		}
	}
	return response.Data.(*mtypes.RPCTransaction), nil
}

func (m *Monaco) callmsgToRequest(msg eth.CallMsg) (*mtypes.ExecutorRequest, uint64) {
	var to *ethcmn.Address
	if msg.To != nil {
		addr := ethcmn.BytesToAddress(msg.To.Bytes())
		to = &addr
	}
	if msg.Value == nil {
		msg.Value = big.NewInt(0)
	}
	var err error
	price := msg.GasPrice
	if price == nil {
		price, err = m.GasPrice()
		if err != nil {
			price = big.NewInt(defaultPrice)
		}
	}
	msg.GasPrice = price

	gas := msg.Gas
	if gas == 0 {
		balance, err := m.GetBalance(msg.From, 0)
		if err != nil {
			balance = big.NewInt(0)
		}
		maxGasLimit := big.NewInt(0)
		maxGasLimit = maxGasLimit.Div(balance, price)
		gas = maxGasLimit.Uint64()
	}
	msg.Gas = gas

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
	return &mtypes.ExecutorRequest{
		Sequences: []*mtypes.ExecutingSequence{
			{
				Msgs: []*eucommon.StandardMessage{
					{
						TxHash: hash,
						Native: &message,
						ID:     0,
					},
				},
				Parallel:   true,
				SequenceId: hash,
			},
		},
		Height:        0,
		GenerationIdx: 0,
		Timestamp:     new(big.Int).SetInt64(time.Now().Unix()),
		Parallelism:   1,
		Debug:         true,
	}, gas
}

func (m *Monaco) Call(msg eth.CallMsg) ([]byte, error) {
	// try run
	var response core.ExecutionResult
	request, _ := m.callmsgToRequest(msg)
	err := intf.Router.Call("estimate-executor", "ExecTxs", request, &response)
	if err != nil {
		return nil, err
	}
	if response.Err != nil {
		return nil, response.Err
	}
	return response.ReturnData, nil
}

func (m *Monaco) SendRawTransaction(rawTx []byte) (ethcmn.Hash, error) {
	var response mtypes.RawTransactionReply
	err := intf.Router.Call("gateway", "SendRawTransaction", &mtypes.RawTransactionArgs{
		Tx: rawTx,
	}, &response)
	if err != nil {
		return ethcmn.Hash{}, err
	}
	return response.TxHash.(ethcmn.Hash), nil
}
func (m *Monaco) SendRawTransactions(rawTxs [][]byte) (uint64, error) {
	var response mtypes.SendTransactionReply
	err := intf.Router.Call("gateway", "ReceivedTransactions", &mtypes.SendTransactionArgs{
		Txs: rawTxs,
	}, &response)
	if err != nil {
		return 0, err
	}
	return uint64(len(rawTxs)), nil
}

func (m *Monaco) GetTransactionReceipt(hash ethcmn.Hash) (*types.Receipt, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Receipt_Eth,
		Data:      hash,
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.(*types.Receipt), nil
}

func (m *Monaco) GetBlockReceipts(height uint64) ([]*types.Receipt, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Block_Receipts,
		Data:      height,
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.([]*types.Receipt), nil
}

func (m *Monaco) GetLogs(filter eth.FilterQuery) ([]*types.Log, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Logs,
		Data:      &filter,
	}, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.([]*types.Log), nil
}

func (m *Monaco) GetTransactionByBlockHashAndIndex(hash ethcmn.Hash, index int) (*mtypes.RPCTransaction, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TxByHashAndIdx,
		Data: &mtypes.RequestBlockEth{
			Hash:  hash,
			Index: index,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	if response.Data == nil {
		return nil, nil
	}
	return response.Data.(*mtypes.RPCTransaction), nil
}
func (m *Monaco) GetTransactionByBlockNumberAndIndex(number int64, index int) (*mtypes.RPCTransaction, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TxByNumberAndIdx,
		Data: &mtypes.RequestBlockEth{
			Number: number,
			Index:  index,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	if response.Data == nil {
		return nil, nil
	}
	return response.Data.(*mtypes.RPCTransaction), nil
}

func (m *Monaco) GetBlockTransactionCountByHash(hash ethcmn.Hash) (int, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TxNumsByHash,
		Data:      hash,
	}, &response)
	if err != nil {
		return 0, err
	}
	return response.Data.(int), nil
}
func (m *Monaco) GetBlockTransactionCountByNumber(number int64) (int, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TxNumsByNumber,
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
	var response mtypes.QueryResult
	err := intf.Router.Call("consensus", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Syncing,
	}, &response)
	if err != nil {
		return false, err
	}
	return response.Data.(bool), nil
}

func (m *Monaco) Proposer() (bool, error) {
	var response mtypes.QueryResult
	err := intf.Router.Call("consensus", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Proposer,
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
