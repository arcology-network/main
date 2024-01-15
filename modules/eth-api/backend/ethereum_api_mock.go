package backend

import (
	"bytes"
	"math/big"
	"sync"

	"github.com/arcology-network/common-lib/types"
	ccdb "github.com/arcology-network/concurrenturl/storage"
	mtypes "github.com/arcology-network/main/types"
	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/engine"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtyp "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/params"
	ethrlp "github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
)

type EthereumAPIMock struct {
	chainID     *big.Int
	blockHeader *ethtyp.Header
	blockGuard  sync.RWMutex
}

func NewEthereumAPIMock(chainID *big.Int) EthereumAPI {
	return &EthereumAPIMock{
		chainID: chainID,
		blockHeader: &ethtyp.Header{
			ParentHash: ethcmn.HexToHash("1234567890123456789012345678901234567890123456789012345678901234"),
			Coinbase:   ethcmn.HexToAddress("0000000000000000000000000000000000000001"),
			Difficulty: new(big.Int).SetUint64(1),
			Number:     new(big.Int).SetUint64(1),
			GasLimit:   0xffffffff,
			GasUsed:    0xffff,
			Extra:      []byte{1},
			MixDigest:  ethcmn.HexToHash("1234567890123456789012345678901234567890123456789012345678901234"),
		},
	}
}

func (mock *EthereumAPIMock) GetProof(rq *types.RequestProof) (*ccdb.AccountResult, error) {
	return &ccdb.AccountResult{}, nil
}

func (mock *EthereumAPIMock) ForkchoiceUpdatedV2(update engine.ForkchoiceStateV1, payloadAttributes *engine.PayloadAttributes, chainid uint64) (engine.ForkChoiceResponse, error) {
	return engine.ForkChoiceResponse{}, nil
}
func (mock *EthereumAPIMock) GetPayloadV2(payloadID engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	return nil, nil
}
func (mock *EthereumAPIMock) NewPayloadV2(params engine.ExecutableData) (engine.PayloadStatusV1, error) {
	return engine.PayloadStatusV1{}, nil
}
func (mock *EthereumAPIMock) SignalSuperchainV1(signal *catalyst.SuperchainSignal) (params.ProtocolVersion, error) {
	return params.ProtocolVersion{}, nil
}

func (mock *EthereumAPIMock) BlockNumber() (uint64, error) {
	mock.blockGuard.RLock()
	defer mock.blockGuard.RUnlock()

	return mock.blockHeader.Number.Uint64(), nil
}

func (mock *EthereumAPIMock) GetBlockByNumber(number int64, fullTx bool) (*mtypes.RPCBlock, error) {
	mock.blockGuard.Lock()
	defer mock.blockGuard.Unlock()

	var header *ethtyp.Header
	if number == mtypes.BlockNumberLatest {
		mock.blockHeader.Number.Add(mock.blockHeader.Number, new(big.Int).SetUint64(1))
		header = mock.blockHeader
	} else {
		header = ethtyp.CopyHeader(mock.blockHeader)
		header.Number.SetInt64(number)
	}

	return &mtypes.RPCBlock{
		Header: ethtyp.CopyHeader(header),
	}, nil
}

func (mock *EthereumAPIMock) GetBlockByHash(hash ethcmn.Hash, fullTx bool) (*mtypes.RPCBlock, error) {
	// TODO
	return nil, nil
}

func (mock *EthereumAPIMock) GetHeaderByHash(hash ethcmn.Hash) (*mtypes.RPCBlock, error) {
	// TODO
	return nil, nil
}
func (mock *EthereumAPIMock) GetHeaderByNumber(number int64) (*mtypes.RPCBlock, error) {
	// TODO
	return nil, nil
}

func (mock *EthereumAPIMock) GetCode(address ethcmn.Address, number int64) ([]byte, error) {
	if bytes.Equal(address.Bytes(), ethcmn.HexToAddress("0x608060405234801561001057600080fd5b503360").Bytes()) {
		return []byte{0xff, 0xff}, nil
	} else if bytes.Equal(address.Bytes(), ethcmn.HexToAddress("0x60c3610025600b82828239805160001a60731461").Bytes()) {
		return []byte{0xee, 0xee}, nil
	} else if bytes.Equal(address.Bytes(), ethcmn.HexToAddress("0x608060405234801561001057600080fd5b506127").Bytes()) {
		return []byte{0xdd, 0xdd}, nil
	}
	return nil, nil
}

func (mock *EthereumAPIMock) GetBalance(address ethcmn.Address, number int64) (*big.Int, error) {
	balance, _ := new(big.Int).SetString("10000000000000000", 0)
	return balance, nil
}

func (mock *EthereumAPIMock) GetTransactionCount(address ethcmn.Address, number int64) (uint64, error) {
	return 0, nil
}

func (mock *EthereumAPIMock) GetStorageAt(address ethcmn.Address, key string, number int64) ([]byte, error) {
	return ethcmn.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000").Bytes(), nil
}

func (mock *EthereumAPIMock) EstimateGas(msg eth.CallMsg) (uint64, error) {
	return 0x1000, nil
}

func (mock *EthereumAPIMock) GasPrice() (*big.Int, error) {
	return new(big.Int).SetUint64(0xff), nil
}

func (mock *EthereumAPIMock) GetTransactionByHash(hash ethcmn.Hash) (*mtypes.RPCTransaction, error) {
	mock.blockGuard.RLock()
	defer mock.blockGuard.RUnlock()

	ti := hexutil.Uint64(uint64(0))
	return &mtypes.RPCTransaction{
		From:             ethcmn.HexToAddress("0x57de3b28c55095e5ca67a8e20fa9d7d5d9aef891"),
		Gas:              0x1400,
		GasPrice:         (*hexutil.Big)(big.NewInt(256)),
		BlockNumber:      (*hexutil.Big)(mock.blockHeader.Number), //new(big.Int).Set(mock.blockHeader.Number),
		TransactionIndex: &ti,
		Value:            (*hexutil.Big)(big.NewInt(0)),
	}, nil
}

func (mock *EthereumAPIMock) Call(msg eth.CallMsg) ([]byte, error) {
	if bytes.Equal(msg.Data[:4], []byte{0xf8, 0xb2, 0xcb, 0x4f}) {
		return ethcmn.BytesToHash([]byte{0x27, 0x10}).Bytes(), nil
	} else if bytes.Equal(msg.Data[:4], []byte{0x7b, 0xd7, 0x03, 0xe8}) {
		return ethcmn.BytesToHash([]byte{0x4e, 0x20}).Bytes(), nil
	}
	return nil, nil
}

func (mock *EthereumAPIMock) SendRawTransaction(rawTx []byte) (ethcmn.Hash, error) {
	tx := new(ethtyp.Transaction)
	ethrlp.DecodeBytes(rawTx, tx)
	msg, _ := core.TransactionToMessage(tx, ethtyp.NewLondonSigner(mock.chainID), nil)
	return ethcmn.BytesToHash(msg.Data[:32]), nil
}

func (mock *EthereumAPIMock) GetTransactionReceipt(hash ethcmn.Hash) (*ethtyp.Receipt, error) {
	mock.blockGuard.RLock()
	defer mock.blockGuard.RUnlock()

	return &ethtyp.Receipt{
		TxHash:           hash,
		TransactionIndex: 0,
		BlockHash:        ethcmn.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"),
		BlockNumber:      new(big.Int).Set(mock.blockHeader.Number),
		GasUsed:          0x1000,
		Status:           1,
		ContractAddress:  ethcmn.BytesToAddress(hash.Bytes()[:20]),
	}, nil
}

func (mock *EthereumAPIMock) GetBlockReceipts(height uint64) ([]*ethtyp.Receipt, error) {
	mock.blockGuard.RLock()
	defer mock.blockGuard.RUnlock()

	return nil, nil
}

func (mock *EthereumAPIMock) GetLogs(filter eth.FilterQuery) ([]*ethtyp.Log, error) {
	return []*ethtyp.Log{}, nil
}

func (mock *EthereumAPIMock) GetTransactionByBlockHashAndIndex(hash ethcmn.Hash, index int) (*mtypes.RPCTransaction, error) {
	mock.blockGuard.RLock()
	defer mock.blockGuard.RUnlock()

	ti := hexutil.Uint64(uint64(0))
	return &mtypes.RPCTransaction{
		From:             ethcmn.HexToAddress("0x57de3b28c55095e5ca67a8e20fa9d7d5d9aef891"),
		Gas:              0x1400,
		GasPrice:         (*hexutil.Big)(big.NewInt(256)),
		BlockNumber:      (*hexutil.Big)(mock.blockHeader.Number), //new(big.Int).Set(mock.blockHeader.Number),
		TransactionIndex: &ti,
		Value:            (*hexutil.Big)(big.NewInt(0)),
	}, nil
}
func (mock *EthereumAPIMock) GetTransactionByBlockNumberAndIndex(number int64, index int) (*mtypes.RPCTransaction, error) {
	mock.blockGuard.RLock()
	defer mock.blockGuard.RUnlock()

	ti := hexutil.Uint64(uint64(0))
	return &mtypes.RPCTransaction{
		From:             ethcmn.HexToAddress("0x57de3b28c55095e5ca67a8e20fa9d7d5d9aef891"),
		Gas:              0x1400,
		GasPrice:         (*hexutil.Big)(big.NewInt(256)),
		BlockNumber:      (*hexutil.Big)(mock.blockHeader.Number), //new(big.Int).Set(mock.blockHeader.Number),
		TransactionIndex: &ti,
		Value:            (*hexutil.Big)(big.NewInt(0)),
	}, nil
}

func (mock *EthereumAPIMock) GetBlockTransactionCountByHash(hash ethcmn.Hash) (int, error) {
	mock.blockGuard.RLock()
	defer mock.blockGuard.RUnlock()

	return 0, nil
}
func (mock *EthereumAPIMock) GetBlockTransactionCountByNumber(number int64) (int, error) {
	mock.blockGuard.RLock()
	defer mock.blockGuard.RUnlock()

	return 0, nil
}
func (mock *EthereumAPIMock) GetUncleCountByBlockHash(hash ethcmn.Hash) (int, error) {
	return 0x0, nil
}
func (mock *EthereumAPIMock) GetUncleCountByBlockNumber(number int64) (int, error) {
	return 0x0, nil
}
func (mock *EthereumAPIMock) SubmitWork() (bool, error) {
	return true, nil
}
func (mock *EthereumAPIMock) SubmitHashrate() (bool, error) {
	return true, nil
}
func (mock *EthereumAPIMock) Hashrate() (int, error) {
	return 0x3e6, nil
}
func (mock *EthereumAPIMock) GetWork() ([]string, error) {
	return []string{}, nil
}
func (mock *EthereumAPIMock) ProtocolVersion() (int, error) {
	return 10000 + 2, nil
}
func (mock *EthereumAPIMock) Syncing() (bool, error) {
	return false, nil
}
func (mock *EthereumAPIMock) Proposer() (bool, error) {
	return false, nil
}

func (mock *EthereumAPIMock) NewFilter(filter eth.FilterQuery) (ID, error) {
	return NewID(), nil
}
func (mock *EthereumAPIMock) NewBlockFilter() (ID, error) {
	return NewID(), nil
}
func (mock *EthereumAPIMock) NewPendingTransactionFilter() (ID, error) {
	return NewID(), nil
}
func (mock *EthereumAPIMock) UninstallFilter(id ID) (bool, error) {
	return true, nil
}
func (mock *EthereumAPIMock) GetFilterChanges(id ID) (interface{}, error) {
	return nil, nil
}
func (mock *EthereumAPIMock) GetFilterLogs(id ID) ([]*ethtyp.Log, error) {
	return []*ethtyp.Log{}, nil
}
