package storage

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	ethRlp "github.com/HPISTechnologies/3rd-party/eth/rlp"
	ethTypes "github.com/HPISTechnologies/3rd-party/eth/types"
	"github.com/HPISTechnologies/common-lib/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/component-lib/ethrpc"
	intf "github.com/HPISTechnologies/component-lib/interface"
	"github.com/HPISTechnologies/component-lib/log"
	evm "github.com/HPISTechnologies/evm"
	evmCommon "github.com/HPISTechnologies/evm/common"
	evmTypes "github.com/HPISTechnologies/evm/core/types"
	mstypes "github.com/HPISTechnologies/main/modules/storage/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

var (
	receiptRequest = promauto.NewSummary(prometheus.SummaryOpts{
		Name: "storage_receipt_request_process_seconds",
		Help: "The response time of the receipt request.",
	})

	storageSingleton actor.IWorkerEx
	initOnce         sync.Once
)

type Storage struct {
	actor.WorkerThread
	caches       *mstypes.LogCaches
	scanCache    *mstypes.ScanCache
	cacheSvcPort string
	lastHeight   uint64
	chainID      *big.Int
}

//return a Subscriber struct
func NewStorage(concurrency int, groupid string) actor.IWorkerEx {
	initOnce.Do(func() {
		storageSingleton = &Storage{}
		storageSingleton.(*Storage).Set(concurrency, groupid)
	})
	return storageSingleton
}

func (s *Storage) Inputs() ([]string, bool) {
	return []string{
		//actor.MsgUrlUpdate,
		actor.MsgBlockCompleted,
		actor.MsgParentInfo,
		actor.MsgSelectedReceipts,
		actor.MsgPendingBlock,
		actor.MsgExecTime,
		actor.MsgSpawnedRelations,
		actor.MsgConflictInclusive,
	}, true
}

func (s *Storage) Outputs() map[string]int {
	return map[string]int{}
}

func (s *Storage) Config(params map[string]interface{}) {
	mstypes.CreateDB(params)
	s.caches = mstypes.NewLogCaches(int(params["log_cache_size"].(float64)))
	s.scanCache = mstypes.NewScanCache(
		int(params["block_cache_size"].(float64)),
		int(params["tx_cache_size"].(float64)),
	)
	s.cacheSvcPort = params["cache_svc_port"].(string)
	s.chainID = params["chain_id"].(*big.Int)
}

func (s *Storage) OnStart() {
	var na int
	intf.Router.Call("statestore", "GetHeight", &na, &s.lastHeight)
	c := cors.AllowAll()
	go http.ListenAndServe(":"+s.cacheSvcPort, c.Handler(NewHandler(s.scanCache)))
}

func (*Storage) Stop() {}

func (s *Storage) OnMessageArrived(msgs []*actor.Message) error {
	//var statedatas *storage.UrlUpdate
	result := ""
	height := uint64(0)
	var receipts []*ethTypes.Receipt
	var block *types.MonacoBlock
	var exectime *types.StatisticalInformation
	spawnedRelations := []*types.SpawnedRelation{}
	inclusive := &types.InclusiveList{}
	var na int

	for _, v := range msgs {
		switch v.Name {
		// case actor.MsgUrlUpdate:
		// 	height = v.Height
		// 	statedatas = v.Data.(*storage.UrlUpdate)
		case actor.MsgBlockCompleted:
			result = v.Data.(string)
		case actor.MsgParentInfo:
			parentinfo := v.Data.(*types.ParentInfo)
			isnil, err := s.IsNil(parentinfo, "parentinfo")
			if isnil {
				return err
			}
			intf.Router.Call("statestore", "Save", &State{
				Height:     v.Height,
				ParentHash: parentinfo.ParentHash,
				ParentRoot: parentinfo.ParentRoot,
			}, &na)
		case actor.MsgSelectedReceipts:
			for _, item := range v.Data.([]interface{}) {
				receipts = append(receipts, item.(*ethTypes.Receipt))
			}
		case actor.MsgPendingBlock:
			block = v.Data.(*types.MonacoBlock)
			height = v.Height
		case actor.MsgExecTime:
			exectime = v.Data.(*types.StatisticalInformation)
		case actor.MsgSpawnedRelations:
			spawnedRelations = v.Data.([]*types.SpawnedRelation)
		case actor.MsgConflictInclusive:
			inclusive = v.Data.(*types.InclusiveList)
		}
	}

	if actor.MsgBlockCompleted_Success == result {
		savet := time.Now()
		s.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>> storage start gather info", zap.Uint64("blockNo", height))

		// if statedatas != nil {
		// 	t := time.Now()
		// 	intf.Router.Call("urlstore", "Save", &UrlSaveRequest{
		// 		Keys:          statedatas.Keys,
		// 		EncodedValues: statedatas.EncodedValues,
		// 	}, &na)
		// 	s.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>> url commit", zap.Duration("time", time.Since(t)), zap.Int("paths", len(statedatas.Keys)))
		// }

		if block != nil && block.Height > 0 {
			t0 := time.Now()
			intf.Router.Call("blockstore", "Save", block, &na)
			s.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>> block save", zap.Duration("time", time.Since(t0)))
			s.scanCache.BlockReceived(block)
		}

		t0 := time.Now()
		relations := map[ethCommon.Hash]ethCommon.Hash{}
		for _, relation := range spawnedRelations {
			relations[relation.Txhash] = relation.SpawnedTxHash
		}

		conflictTxs := map[ethCommon.Hash]int{}
		if inclusive != nil {
			for i, hash := range inclusive.HashList {
				if !inclusive.Successful[i] {
					conflictTxs[*hash] = i
				}
			}
		}
		blockHash := block.Hash()
		failed := 0
		keys := make([]string, len(receipts))
		worker := func(start, end int, idx int, args ...interface{}) {
			for i := start; i < end; i++ {
				txhash := receipts[i].TxHash
				if spawnedHash, ok := relations[txhash]; ok {
					receipts[i].SpawnedTxHash = spawnedHash
				}

				if _, ok := conflictTxs[txhash]; ok {
					receipts[i].Status = 0
					receipts[i].GasUsed = 0
				}

				if receipts[i].Status == 0 {
					failed = failed + 1
				}

				receipts[i].BlockHash = ethCommon.BytesToHash(blockHash)
				receipts[i].BlockNumber = big.NewInt(int64(block.Height))
				receipts[i].TransactionIndex = uint(i)

				for k := range receipts[i].Logs {
					receipts[i].Logs[k].BlockHash = receipts[i].BlockHash
					receipts[i].Logs[k].TxHash = receipts[i].TxHash
					receipts[i].Logs[k].TxIndex = receipts[i].TransactionIndex
				}
				keys[i] = string(receipts[i].TxHash.Bytes())
			}
		}
		common.ParallelWorker(len(receipts), s.Concurrency, worker)

		if len(receipts) > 0 {
			intf.Router.Call("receiptstore", "Save", &SaveReceiptsRequest{
				Height:   height,
				Receipts: receipts,
			}, &na)
			intf.Router.Call("indexerstore", "Save", &SaveIndexRequest{
				Height: height,
				keys:   keys,
				IsSave: true,
			}, &na)
			intf.Router.Call("indexerstore", "SaveBlockHash", &SaveIndexBlockHashRequest{
				Height: height,
				Hash:   string(blockHash),
				IsSave: true,
			}, &na)
			s.AddLog(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>> receipt save", zap.Int("total", len(receipts)), zap.Int("failed", failed), zap.Duration("time", time.Since(t0)))
			s.caches.Add(height, receipts)
		}

		if exectime != nil && len(exectime.Key) > 0 {
			intf.Router.Call("debugstore", "SaveStatisticInfos", &StatisticInfoSaveRequest{
				Height:          height,
				StatisticalInfo: exectime,
			}, &na)
		}

		s.AddLog(log.LogLevel_Info, "<<<<<<<<<<<<<<<<<<<<< storage gather info completed", zap.Duration("save time", time.Since(savet)), zap.Uint64("blockNo", height))
		s.lastHeight = height
	}

	return nil
}

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
		response.Data = &block
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
			var header ethTypes.Header
			err := ethRlp.DecodeBytes(data[1:], &header)

			if err != nil {
				rs.AddLog(log.LogLevel_Error, "block header decode err", zap.String("err", err.Error()))
				return err
			}
			coinbase = header.Coinbase.String()
			gasUsed = big.NewInt(int64(header.GasUsed))
			hash = fmt.Sprintf("%x", header.Hash())
			timestamp = int(header.Time.Int64())
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
		request := request.Data.(types.RequestContainer)
		key := string(ethCommon.Hex2Bytes(request.Key))
		data := []byte{}
		var containerType int
		switch request.Style {
		case types.ConcurrentLibStyle_Array:
			containerType = ContainerTypeArray
		case types.ConcurrentLibStyle_Map:
			containerType = ContainerTypeMap
		case types.ConcurrentLibStyle_Queue:
			containerType = ContainerTypeQueue
		}
		intf.Router.Call("urlstore", "GetContainerElem", &UrlContainerGetRequest{
			Address:       request.Address,
			Id:            request.Id,
			ContainerType: containerType,
			Key:           key,
		}, &data)
		response.Data = data
	case types.QueryType_Receipt:
		start := time.Now()
		requestReceipt := request.Data.(*types.RequestReceipt)
		if requestReceipt == nil {
			return nil
		}
		hashes := requestReceipt.Hashes

		receipts := make([]*types.Receipt, 0, len(hashes))
		for _, hash := range hashes {
			txhash := ethCommon.HexToHash(hash)
			txhashstr := string(txhash.Bytes())
			var position *mstypes.Position
			intf.Router.Call("indexerstore", "GetPosition", &txhashstr, &position)
			if position == nil {
				continue
			}
			var receipt *ethTypes.Receipt
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

			receiptNew := &types.Receipt{
				Status:          int(receipt.Status),
				ContractAddress: fmt.Sprintf("%x", receipt.ContractAddress),
				GasUsed:         big.NewInt(int64(receipt.GasUsed)),
				Logs:            logs,
				Height:          int(receipt.BlockNumber.Int64()),
				SpawnedTxHash:   fmt.Sprintf("%x", receipt.SpawnedTxHash),
			}
			if requestReceipt.ExecutingDebugLogs {
				intf.Router.Call("debugstore", "GetExecLog", &txhash, &receiptNew.ExecutingLogs)
			}
			receipts = append(receipts, receiptNew)

		}

		receiptRequest.Observe(time.Since(start).Seconds())
		response.Data = receipts

	//------------------------------------------------for Ethereum rpc api-----------------------------------------
	case types.QueryType_BlockNumber:
		response.Data = rs.lastHeight
	case types.QueryType_TransactionCount:
		request := request.Data.(*types.RequestParameters)
		address := fmt.Sprintf("%x", request.Address.Bytes())
		var nonce uint64
		intf.Router.Call("urlstore", "GetNonce", &address, &nonce)
		response.Data = nonce
	case types.QueryType_Code:
		request := request.Data.(*types.RequestParameters)
		address := fmt.Sprintf("%x", request.Address.Bytes())
		var code []byte
		intf.Router.Call("urlstore", "GetCode", &address, &code)
		response.Data = code
	case types.QueryType_Balance_Eth:
		request := request.Data.(*types.RequestParameters)
		address := fmt.Sprintf("%x", request.Address.Bytes())
		var balance *big.Int
		intf.Router.Call("urlstore", "GetBalance", &address, &balance)
		response.Data = balance
	case types.QueryType_Storage:
		request := request.Data.(*types.RequestStorage)
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
		var receipt *ethTypes.Receipt
		intf.Router.Call("receiptstore", "Get", position, &receipt)
		if receipt == nil {
			response.Data = nil
			return errors.New("receipt not found")
		}

		// var receipt *ethTypes.Receipt
		// intf.Router.Call("receiptstore", "GetByHash", &hash, &receipt)
		// if receipt == nil {
		// 	response.Data = nil
		// 	return errors.New("receipt not found")
		// }
		receiptRet := evmTypes.Receipt{
			Type:              receipt.Type,
			PostState:         receipt.PostState,
			Status:            receipt.Status,
			CumulativeGasUsed: receipt.CumulativeGasUsed,
			Bloom:             evmTypes.Bloom(receipt.Bloom),
			TxHash:            evmCommon.Hash(receipt.TxHash),
			ContractAddress:   evmCommon.Address(receipt.ContractAddress),
			GasUsed:           receipt.GasUsed,
			BlockHash:         evmCommon.Hash(receipt.BlockHash),
			BlockNumber:       receipt.BlockNumber,
			TransactionIndex:  receipt.TransactionIndex,
		}
		logs := make([]*evmTypes.Log, len(receipt.Logs))
		for i, log := range receipt.Logs {
			topics := make([]evmCommon.Hash, len(log.Topics))
			for j, topic := range log.Topics {
				topics[j] = evmCommon.BytesToHash(topic.Bytes())
			}
			logs[i] = &evmTypes.Log{
				Address:     evmCommon.BytesToAddress(log.Address.Bytes()),
				Topics:      topics,
				Data:        log.Data,
				BlockNumber: log.BlockNumber,
				TxHash:      evmCommon.BytesToHash(log.TxHash.Bytes()),
				TxIndex:     log.TxIndex,
				BlockHash:   evmCommon.BytesToHash(log.BlockHash.Bytes()),
				Index:       log.Index,
				Removed:     log.Removed,
			}
		}
		receiptRet.Logs = logs
		response.Data = &receiptRet
	case types.QueryType_Transaction:
		hash := request.Data.(evmCommon.Hash)
		txhashstr := string(hash.Bytes())
		var position *mstypes.Position
		intf.Router.Call("indexerstore", "GetPosition", &txhashstr, &position)
		if position == nil {
			response.Data = nil
			return errors.New("hash not found")
		}

		transaction, err := rs.getTransaction(position.Height, position.IdxInBlock)
		if err != nil {
			return err
		}
		response.Data = transaction
	case types.QueryType_Block_Eth:
		request := request.Data.(*types.RequestBlockEth)
		queryHeight := rs.getQueryHeight(request.Number)

		rpcBlock, err := rs.getRpcBlock(queryHeight, request.FullTx)
		if err != nil {
			return err
		}
		response.Data = rpcBlock
	case types.QueryType_BlocByHash:
		request := request.Data.(*types.RequestBlockEth)
		hash := string(request.Hash.Bytes())
		var height uint64
		intf.Router.Call("indexerstore", "GetHeightByHash", &hash, &height)
		rpcBlock, err := rs.getRpcBlock(height, request.FullTx)
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
		transaction, err := rs.getTransaction(height, request.Index)
		if err != nil {
			return err
		}
		response.Data = transaction
	case types.QueryType_TxByNumberAndIdx:
		request := request.Data.(*types.RequestBlockEth)
		height := rs.getQueryHeight(request.Number)
		transaction, err := rs.getTransaction(height, request.Index)
		if err != nil {
			return err
		}
		response.Data = transaction
	}

	return nil
}
func (rs *Storage) getTransaction(height uint64, idx int) (*ethrpc.RPCTransaction, error) {
	var receipt *ethTypes.Receipt
	position := mstypes.Position{
		Height:     height,
		IdxInBlock: idx,
	}
	intf.Router.Call("receiptstore", "Get", &position, &receipt)
	var tx *ethTypes.Transaction
	intf.Router.Call("blockstore", "GetTransaction", &position, &tx)

	msg, err := tx.AsMessage(ethTypes.NewEIP155Signer(rs.chainID))
	if err != nil {
		return nil, err
	}
	transactionIndex := uint64(receipt.TransactionIndex)
	v, s, r := tx.RawSignatureValues()
	return &ethrpc.RPCTransaction{
		BlockHash:        evmCommon.Hash(receipt.BlockHash),
		BlockNumber:      receipt.BlockNumber,
		From:             evmCommon.Address(msg.From()),
		Gas:              receipt.GasUsed,
		GasPrice:         tx.GasPrice(),
		Hash:             evmCommon.Hash(receipt.TxHash),
		Input:            msg.Data(),
		Nonce:            tx.Nonce(),
		To:               (*evmCommon.Address)(msg.To()),
		TransactionIndex: &transactionIndex,
		Value:            msg.Value(),
		Type:             uint64(receipt.Type),
		V:                v,
		S:                s,
		R:                r,
	}, nil
}
func (rs *Storage) getQueryHeight(number int64) uint64 {
	queryHeight := uint64(0)
	if number < 0 {
		switch number {
		case ethrpc.BlockNumberLatest:
			queryHeight = rs.lastHeight
		case ethrpc.BlockNumberPending:
			queryHeight = rs.lastHeight
		case ethrpc.BlockNumberEarliest:
			queryHeight = uint64(0)
		default:
			queryHeight = rs.lastHeight
		}
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
func (rs *Storage) getRpcBlock(height uint64, fulltx bool) (*ethrpc.RPCBlock, error) {
	var block *types.MonacoBlock
	intf.Router.Call("blockstore", "GetByHeight", &height, &block)

	header := evmTypes.Header{}
	for i := range block.Headers {
		if block.Headers[i][0] != types.AppType_Eth {
			continue
		}

		ethheader := ethTypes.Header{}
		err := ethRlp.DecodeBytes(block.Headers[i][1:], &ethheader)
		if err != nil {
			rs.AddLog(log.LogLevel_Error, "block header decode err", zap.String("err", err.Error()))
			return nil, err
		}
		header = evmTypes.Header{
			ParentHash:  evmCommon.Hash(ethheader.ParentHash),
			Number:      ethheader.Number,
			Time:        ethheader.Time.Uint64(),
			Difficulty:  ethheader.Difficulty,
			Coinbase:    evmCommon.Address(ethheader.Coinbase),
			Root:        evmCommon.Hash(ethheader.Root),
			GasUsed:     ethheader.GasUsed,
			TxHash:      evmCommon.Hash(ethheader.TxHash),
			ReceiptHash: evmCommon.Hash(ethheader.ReceiptHash),
			GasLimit:    ethheader.GasLimit,
		}
	}

	rpcBlock := ethrpc.RPCBlock{
		Header: &header,
	}
	var hashstr []string
	intf.Router.Call("indexerstore", "GetBlockHashes", &block.Height, &hashstr)
	if fulltx {
		transactions := make([]interface{}, len(hashstr))
		for i := range hashstr {
			var tx *ethTypes.Transaction
			intf.Router.Call("blockstore", "GetTxByHash", &mstypes.Position{
				Height:     block.Height,
				IdxInBlock: i,
			}, &tx)
			if tx.To() == nil {
				transactions[i] = evmTypes.NewContractCreation(tx.Nonce(), tx.Value(), tx.Gas(), tx.GasPrice(), tx.Data())
			} else {
				transactions[i] = evmTypes.NewTransaction(tx.Nonce(), evmCommon.Address(*tx.To()), tx.Value(), tx.Gas(), tx.GasPrice(), tx.Data())
			}
		}
		rpcBlock.Transactions = transactions
	} else {
		hashes := make([]interface{}, len(hashstr))
		for i := range hashstr {
			hashes[i] = evmCommon.HexToHash(hashstr[i])
		}
		rpcBlock.Transactions = hashes
	}
	return &rpcBlock, nil
}
