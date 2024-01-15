package storage

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/arcology-network/common-lib/types"
	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	intf "github.com/arcology-network/streamer/interface"
	"github.com/arcology-network/streamer/log"
	evm "github.com/ethereum/go-ethereum"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/zap"
)

func (rs *Storage) Query(ctx context.Context, request *types.QueryRequest, response *types.QueryResult) error {
	switch request.QueryType {
	case types.QueryType_LatestHeight:
		response.Data = int(rs.lastHeight)
	case types.QueryType_Nonce:
		request := request.Data.(types.RequestBalance)
		var nonce uint64
		intf.Router.Call("urlstore", "GetNonce", &request.Address, &nonce)
		response.Data = nonce
	case types.QueryType_Balance:
		request := request.Data.(types.RequestBalance)
		var balance *big.Int
		intf.Router.Call("urlstore", "GetBalance", &request.Address, &balance)
		response.Data = balance
	case types.QueryType_RawBlock:
		queryHeight := request.Data.(uint64)
		var block *types.MonacoBlock
		intf.Router.Call("blockstore", "GetByHeight", &queryHeight, &block)
		if block == nil {
			return fmt.Errorf("block not found for height %d", queryHeight)
		}
		response.Data = block
	case types.QueryType_Block:
		blockRequest := request.Data.(*types.RequestBlock)
		if blockRequest == nil {
			return errors.New("query params is nil")
		}
		var queryHeight uint64
		if blockRequest.Height < 0 {
			queryHeight = rs.lastHeight
		} else {
			queryHeight = uint64(blockRequest.Height)
		}
		var block *types.MonacoBlock
		intf.Router.Call("blockstore", "GetByHeight", &queryHeight, &block)
		if block == nil {
			return errors.New("block is nil")
		}
		var statisticInfo *types.StatisticalInformation
		intf.Router.Call("debugstore", "GetStatisticInfos", &queryHeight, &statisticInfo)
		if statisticInfo == nil {
			statisticInfo = &types.StatisticalInformation{
				TimeUsed: 0,
			}
		}

		coinbase := ""
		gasUsed := big.NewInt(0)
		hash := ""
		timestamp := 0
		height := 0
		if len(block.Headers) > 0 {
			data := block.Headers[0]
			var header evmTypes.Header
			err := header.UnmarshalJSON(data[1:])
			if err != nil {
				rs.AddLog(log.LogLevel_Error, "block header decode err", zap.String("err", err.Error()))
				return err
			}
			coinbase = header.Coinbase.String()
			gasUsed = big.NewInt(int64(header.GasUsed))
			hash = fmt.Sprintf("%x", header.Hash())
			timestamp = int(header.Time)
			height = int(block.Height)
		}

		queryBlock := types.Block{
			Height:    height,
			Hash:      hash,
			Coinbase:  coinbase,
			Number:    len(block.Txs),
			ExecTime:  statisticInfo.TimeUsed,
			GasUsed:   gasUsed,
			Timestamp: timestamp,
		}
		if blockRequest.Transactions {
			var hashes []string
			intf.Router.Call("indexerstore", "GetBlockHashes", &queryHeight, &hashes)
			queryBlock.Transactions = hashes
		}
		response.Data = queryBlock

	case types.QueryType_Container:
		// request := request.Data.(types.RequestContainer)
		// key := string(evmCommon.Hex2Bytes(request.Key))
		data := []byte{}
		// var containerType int
		// switch request.Style {
		// case types.ConcurrentLibStyle_Array:
		// 	containerType = ContainerTypeArray
		// case types.ConcurrentLibStyle_Map:
		// 	containerType = ContainerTypeMap
		// case types.ConcurrentLibStyle_Queue:
		// 	containerType = ContainerTypeQueue
		// }
		// intf.Router.Call("urlstore", "GetContainerElem", &UrlContainerGetRequest{
		// 	Address:       request.Address,
		// 	Id:            request.Id,
		// 	ContainerType: containerType,
		// 	Key:           key,
		// }, &data)
		response.Data = data
	case types.QueryType_Receipt:
		start := time.Now()
		requestReceipt := request.Data.(*types.RequestReceipt)
		if requestReceipt == nil {
			return nil
		}
		hashes := requestReceipt.Hashes

		receipts := make([]*types.QueryReceipt, 0, len(hashes))
		for _, hash := range hashes {
			txhash := evmCommon.HexToHash(hash)
			txhashstr := string(txhash.Bytes())
			var position *mstypes.Position
			intf.Router.Call("indexerstore", "GetPosition", &txhashstr, &position)
			if position == nil {
				continue
			}
			var receipt *evmTypes.Receipt
			intf.Router.Call("receiptstore", "Get", position, &receipt)
			if receipt == nil {
				continue
			}

			logs := make([]*types.Log, len(receipt.Logs))
			for j, log := range receipt.Logs {
				topics := make([]string, len(log.Topics))
				for k, topic := range log.Topics {
					topics[k] = fmt.Sprintf("%x", topic.Bytes())
				}
				logs[j] = &types.Log{
					Address:     fmt.Sprintf("%x", log.Address.Bytes()),
					Topics:      topics,
					Data:        fmt.Sprintf("%x", log.Data),
					BlockNumber: log.BlockNumber,
					TxHash:      fmt.Sprintf("%x", log.TxHash.Bytes()),
					TxIndex:     log.TxIndex,
					BlockHash:   fmt.Sprintf("%x", log.BlockHash.Bytes()),
					Index:       log.Index,
				}
			}

			receiptNew := &types.QueryReceipt{
				Status:          int(receipt.Status),
				ContractAddress: fmt.Sprintf("%x", receipt.ContractAddress),
				GasUsed:         big.NewInt(int64(receipt.GasUsed)),
				Logs:            logs,
				Height:          int(receipt.BlockNumber.Int64()),
			}

			receipts = append(receipts, receiptNew)

		}

		receiptRequest.Observe(time.Since(start).Seconds())
		response.Data = receipts

	//------------------------------------------------for Ethereum rpc api-----------------------------------------
	case types.QueryType_BlockNumber:
		response.Data = rs.lastHeight
	case types.QueryType_TransactionCount:
		request := request.Data.(types.RequestParameters)
		address := fmt.Sprintf("%x", request.Address.Bytes())
		var nonce uint64
		intf.Router.Call("urlstore", "GetNonce", &address, &nonce)
		response.Data = nonce
	case types.QueryType_Code:
		request := request.Data.(types.RequestParameters)
		address := fmt.Sprintf("%x", request.Address.Bytes())
		var code []byte
		intf.Router.Call("urlstore", "GetCode", &address, &code)
		response.Data = code
	case types.QueryType_Balance_Eth:
		request := request.Data.(*types.RequestParameters)
		address := fmt.Sprintf("%x", request.Address.Bytes())
		var balance *big.Int
		err := intf.Router.Call("urlstore", "GetBalance", &address, &balance)
		if err != nil {
			return err
		}
		response.Data = balance
	case types.QueryType_Storage:
		request := request.Data.(types.RequestStorage)
		address := fmt.Sprintf("%x", request.Address.Bytes())
		var value []byte
		intf.Router.Call("urlstore", "GetEthStorage", &UrlEthStorageGetRequest{
			Address: address,
			Key:     request.Key,
		}, &value)
		response.Data = value
	case types.QueryType_Receipt_Eth:
		hash := request.Data.(evmCommon.Hash)
		txhashstr := string(hash.Bytes())
		var position *mstypes.Position
		intf.Router.Call("indexerstore", "GetPosition", &txhashstr, &position)
		if position == nil {
			response.Data = nil
			return errors.New("receipt not found")
		}
		var receipt *evmTypes.Receipt
		intf.Router.Call("receiptstore", "Get", position, &receipt)
		if receipt == nil {
			response.Data = nil
			return errors.New("receipt not found")
		}

		response.Data = receipt
	case types.QueryType_Block_Receipts:
		height := request.Data.(uint64)

		var receipts []*evmTypes.Receipt
		intf.Router.Call("receiptstore", "GetBlockReceipts", height, &receipts)
		if receipts == nil {
			response.Data = nil
			return errors.New("receipts not found")
		}

		response.Data = receipts
	case types.QueryType_Transaction:
		hash := request.Data.(evmCommon.Hash)
		txhashstr := string(hash.Bytes())
		var position *mstypes.Position
		intf.Router.Call("indexerstore", "GetPosition", &txhashstr, &position)
		if position == nil {
			response.Data = nil
			return errors.New("hash not found")
		}
		block, signerType, err := rs.getRpcBlock(position.Height, false, true)
		if err != nil {
			return err
		}
		transaction, err := rs.getTransactionByPosition(block.Header.Hash(), position.Height, uint64(position.IdxInBlock), block.Header.BaseFee, signerType)
		if err != nil {
			return err
		}
		response.Data = transaction
	case types.QueryType_Block_Eth:
		request := request.Data.(*types.RequestBlockEth)
		queryHeight := rs.getQueryHeight(request.Number)

		rpcBlock, _, err := rs.getRpcBlock(queryHeight, request.FullTx, false)
		if err != nil {
			return err
		}
		response.Data = rpcBlock
	case types.QueryType_HeaderByNumber:
		request := request.Data.(*types.RequestBlockEth)
		queryHeight := rs.getQueryHeight(request.Number)

		rpcBlock, _, err := rs.getRpcBlock(queryHeight, false, true)
		if err != nil {
			return err
		}
		response.Data = rpcBlock
	case types.QueryType_HeaderByHash:
		request := request.Data.(*types.RequestBlockEth)
		hash := string(request.Hash.Bytes())
		var height uint64
		intf.Router.Call("indexerstore", "GetHeightByHash", &hash, &height)
		rpcBlock, _, err := rs.getRpcBlock(height, false, true)
		if err != nil {
			return err
		}
		response.Data = rpcBlock
	case types.QueryType_BlocByHash:
		request := request.Data.(*types.RequestBlockEth)
		hash := string(request.Hash.Bytes())
		var height uint64
		intf.Router.Call("indexerstore", "GetHeightByHash", &hash, &height)
		rpcBlock, _, err := rs.getRpcBlock(height, request.FullTx, false)
		if err != nil {
			return err
		}
		response.Data = rpcBlock
	case types.QueryType_Logs:
		request := request.Data.(*evm.FilterQuery)
		response.Data = rs.caches.Query(*request)
	case types.QueryType_TxNumsByHash:
		hash := string(request.Data.(evmCommon.Hash).Bytes())
		var height uint64
		intf.Router.Call("indexerstore", "GetHeightByHash", &hash, &height)
		response.Data = rs.getBlockTxs(height)
	case types.QueryType_TxNumsByNumber:
		number := request.Data.(int64)
		height := rs.getQueryHeight(number)
		response.Data = rs.getBlockTxs(height)
	case types.QueryType_TxByHashAndIdx:
		request := request.Data.(*types.RequestBlockEth)
		hash := string(request.Hash.Bytes())
		var height uint64
		intf.Router.Call("indexerstore", "GetHeightByHash", &hash, &height)
		block, signerType, err := rs.getRpcBlock(height, false, true)
		if err != nil {
			return err
		}
		transaction, err := rs.getTransactionByPosition(block.Header.Hash(), height, uint64(request.Index), block.Header.BaseFee, signerType)
		if err != nil {
			return err
		}
		response.Data = transaction
	case types.QueryType_TxByNumberAndIdx:
		request := request.Data.(*types.RequestBlockEth)
		height := rs.getQueryHeight(request.Number)
		block, signerType, err := rs.getRpcBlock(height, false, true)
		if err != nil {
			return err
		}

		transaction, err := rs.getTransactionByPosition(block.Header.Hash(), height, uint64(request.Index), block.Header.BaseFee, signerType)
		if err != nil {
			return err
		}
		response.Data = transaction
	}

	return nil
}

func (rs *Storage) getTransactionByPosition(blockHash evmCommon.Hash, height uint64, idx uint64, baseFee *big.Int, SignerType uint8) (*mtypes.RPCTransaction, error) {
	position := mstypes.Position{
		Height:     height,
		IdxInBlock: int(idx),
	}

	var tx *evmTypes.Transaction
	intf.Router.Call("blockstore", "GetTransaction", &position, &tx)

	signer := types.MakeSigner(SignerType, rs.chainID)
	from, _ := evmTypes.Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()
	result := &mtypes.RPCTransaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(), //types.RlpHash(tx),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	if blockHash != (evmCommon.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(height))
		result.TransactionIndex = (*hexutil.Uint64)(&idx)
	}
	switch tx.Type() {
	case evmTypes.DepositTxType:
		srcHash := tx.SourceHash()
		isSystemTx := tx.IsSystemTx()
		result.SourceHash = &srcHash
		if isSystemTx {
			// Only include IsSystemTx when true
			result.IsSystemTx = &isSystemTx
		}
		result.Mint = (*hexutil.Big)(tx.Mint())

		var receipt *evmTypes.Receipt
		intf.Router.Call("receiptstore", "Get", &position, &receipt)

		if receipt != nil && receipt.DepositNonce != nil {
			result.Nonce = hexutil.Uint64(*receipt.DepositNonce)
			if receipt.DepositReceiptVersion != nil {
				result.DepositReceiptVersion = new(hexutil.Uint64)
				*result.DepositReceiptVersion = hexutil.Uint64(*receipt.DepositReceiptVersion)
			}
		}
	case evmTypes.LegacyTxType:
		if v.Sign() == 0 && r.Sign() == 0 && s.Sign() == 0 { // pre-bedrock relayed tx does not have a signature
			result.ChainID = (*hexutil.Big)(new(big.Int).Set(rs.chainID))
			break
		}
		// if a legacy transaction has an EIP-155 chain id, include it explicitly
		if id := tx.ChainId(); id.Sign() != 0 {
			result.ChainID = (*hexutil.Big)(id)
		}

	case evmTypes.AccessListTxType:
		al := tx.AccessList()
		yparity := hexutil.Uint64(v.Sign())
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.YParity = &yparity

	case evmTypes.DynamicFeeTxType:
		al := tx.AccessList()
		yparity := hexutil.Uint64(v.Sign())
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.YParity = &yparity
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (evmCommon.Hash{}) {
			// price = min(gasTipCap + baseFee, gasFeeCap)
			result.GasPrice = (*hexutil.Big)(effectiveGasPrice(tx, baseFee))
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}

	case evmTypes.BlobTxType:
		al := tx.AccessList()
		yparity := hexutil.Uint64(v.Sign())
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.YParity = &yparity
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (evmCommon.Hash{}) {
			result.GasPrice = (*hexutil.Big)(effectiveGasPrice(tx, baseFee))
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
		result.MaxFeePerBlobGas = (*hexutil.Big)(tx.BlobGasFeeCap())
		result.BlobVersionedHashes = tx.BlobHashes()
	}
	return result, nil
}

// effectiveGasPrice computes the transaction gas fee, based on the given basefee value.
//
//	price = min(gasTipCap + baseFee, gasFeeCap)
func effectiveGasPrice(tx *evmTypes.Transaction, baseFee *big.Int) *big.Int {
	fee := tx.GasTipCap()
	fee = fee.Add(fee, baseFee)
	if tx.GasFeeCapIntCmp(fee) < 0 {
		return tx.GasFeeCap()
	}
	return fee
}

func (rs *Storage) getQueryHeight(number int64) uint64 {
	queryHeight := uint64(0)
	if number < 0 {
		queryHeight = rs.lastHeight
	} else {
		queryHeight = uint64(number)
	}
	return queryHeight
}
func (rs *Storage) getBlockTxs(height uint64) int {
	var block *types.MonacoBlock
	intf.Router.Call("blockstore", "GetByHeight", &height, &block)
	return len(block.Txs)
}
func (rs *Storage) getRpcBlock(height uint64, fulltx bool, onlyHeader bool) (*mtypes.RPCBlock, uint8, error) {
	var block *types.MonacoBlock
	intf.Router.Call("blockstore", "GetByHeight", &height, &block)

	header := evmTypes.Header{}
	for i := range block.Headers {
		if block.Headers[i][0] != types.AppType_Eth {
			continue
		}

		ethheader := evmTypes.Header{}
		err := ethheader.UnmarshalJSON(block.Headers[i][1:])
		if err != nil {
			rs.AddLog(log.LogLevel_Error, "block header decode err", zap.String("err", err.Error()))
			return nil, 0, err
		}
		header = ethheader
	}

	rpcBlock := mtypes.RPCBlock{
		Header: &header,
	}
	if onlyHeader {
		return &rpcBlock, block.Signer, nil
	}
	if fulltx {
		transactions := make([]interface{}, len(block.Txs))
		for i := range block.Txs {
			rpctransaction, err := rs.getTransactionByPosition(header.Hash(), height, uint64(i), header.BaseFee, block.Signer)
			if err != nil {
				return nil, 0, err
			}
			transactions[i] = rpctransaction
		}
		rpcBlock.Transactions = transactions
	} else {
		var hashstr []string
		intf.Router.Call("indexerstore", "GetBlockHashes", &block.Height, &hashstr)

		hashes := make([]interface{}, len(hashstr))
		for i := range hashstr {
			hashes[i] = evmCommon.HexToHash(hashstr[i])
		}
		rpcBlock.Transactions = hashes
	}
	return &rpcBlock, block.Signer, nil
}
