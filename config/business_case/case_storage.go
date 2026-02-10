package businessCase

import (
	"fmt"
	"math/big"
	"time"

	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/main/modules/storage"
	queryplan "github.com/arcology-network/main/modules/storage/query_plan"
	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

const (
	MsgStorageTestOk = "StorageOK"
)

type StorageTest struct {
	basePath string
	msgs     []string
	sender   actor.OutboundSender
	from     string
}

func NewStorageTest(basePath string) *StorageTest {
	handler := &StorageTest{
		basePath: basePath,
		msgs:     []string{},
		from:     "StorageTest",
	}
	return handler
}

func (st *StorageTest) SetSender(sender actor.OutboundSender) {
	st.sender = sender
}

func (st *StorageTest) Inputs() ([]string, bool) {
	return []string{
		MsgStorageTestOk,
	}, false
}

func (st *StorageTest) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgParentInfo:        1,
		scommon.MsgSelectedReceipts:  1,
		scommon.MsgPendingBlock:      1,
		scommon.MsgConflictInclusive: 1,
	}
}

func (st *StorageTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(MsgStorageTestOk, st.receivedMsgs)
}

func (st *StorageTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name), logger.F("height", msg.Data.(uint64)))
	st.msgs = append(st.msgs, msg.Name)
	return nil
}

func (st *StorageTest) startTest(ss *broker.StatefulStreamer) []string {
	block, _ := MakeMonacoBlock()
	m := scommon.NewMessageForStream(scommon.MsgPendingBlock, block)
	m.Height = 10
	ss.Send(scommon.MsgPendingBlock, m)
	// time.Sleep(1 * time.Second)
	receipts := MakeReceipts()
	m = scommon.NewMessageForStream(scommon.MsgSelectedReceipts, receipts)
	m.Height = 10
	ss.Send(scommon.MsgSelectedReceipts, m)
	// time.Sleep(1 * time.Second)

	p := mtypes.ParentInfo{
		ParentRoot:    evmCommon.BytesToHash([]byte{1, 2, 3, 4, 5, 6}),
		ParentHash:    evmCommon.BytesToHash([]byte{11, 12, 13, 14, 15, 61}),
		ExcessBlobGas: 788333,
		BlobGasUsed:   456454234,
	}
	m = scommon.NewMessageForStream(scommon.MsgParentInfo, &p)
	m.Height = 10
	ss.Send(scommon.MsgParentInfo, m)
	// time.Sleep(1 * time.Second)

	hs := make([]evmCommon.Hash, len(receipts))
	successfuls := make([]bool, len(receipts))
	for i := range hs {
		hs[i] = receipts[i].TxHash
		successfuls[i] = true
	}

	list := types.InclusiveList{
		HashList:   hs,
		Successful: successfuls,
	}
	m = scommon.NewMessageForStream(scommon.MsgConflictInclusive, &list)
	m.Height = 10
	ss.Send(scommon.MsgConflictInclusive, m)
	time.Sleep(1 * time.Second)

	return st.msgs
}

func (st *StorageTest) startTestQueryBase(ss *broker.StatefulStreamer) []string {
	st.msgs = []string{}

	blockHashNotExist := evmCommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000") // evmCommon.BytesToHash([]byte{1, 2, 3, 45, 6, 7, 8, 90, 11, 111, 122, 33})
	// h, err := st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
	// 	QueryType: mtypes.QueryType_TestHeightByHashOrNumber,
	// 	Data: &mtypes.BlockNumberOrHash{
	// 		// BlockNumber: big.NewInt(10),
	// 		BlockHash: &blockHashNotExist,
	// 	},
	// }, 1, st.from)

	// if err != nil {
	// 	fmt.Printf("******storage.Query err:%v\n", err)
	// 	return st.msgs
	// }
	// fmt.Printf("-------storage.Query height:%v\n", h.(*mtypes.QueryResult).Data.(*big.Int))
	// st.msgs = append(st.msgs, "QueryType_TestHeightByHashOrNumber")

	// QueryType_Block_Receipts
	h, err := st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestHeightByHashOrNumber,
		Data: &mtypes.BlockNumberOrHash{
			// BlockNumber: big.NewInt(0),
			BlockHash: &blockHashNotExist,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query receipts:%v\n", h.(*mtypes.QueryResult).Data.([]*evmTypes.Receipt))
	st.msgs = append(st.msgs, "QueryType_Block_Receipts")

	return st.msgs
}

func (st *StorageTest) startTestQueryRpcBase(ss *broker.StatefulStreamer) []string {
	st.msgs = []string{}

	//----------------------------------------------
	_, store, _ := MakeStateStore(st.basePath)

	_, err := st.sender.SendSync("urlstore", "Init", store.ReadOnlyStore(), 1, st.from)
	if err != nil {
		fmt.Printf("******urlstore.Init err:%v\n", err)
		return st.msgs
	}
	st.msgs = append(st.msgs, "urlstore.Init")

	//----------------------
	bal, err := st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Balance_Eth,
		Data: &mtypes.RequestParameters{
			Address: evmCommon.HexToAddress("0x9f9E0F23aFd5404b34006678c900629183c9A25d"),
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query balance:%v\n", bal.(*mtypes.QueryResult).Data.(*big.Int))
	st.msgs = append(st.msgs, "QueryType_Balance_Eth")

	//----------------------
	bal, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TransactionCount,
		Data: &mtypes.RequestParameters{
			Address: evmCommon.HexToAddress("0x9f9E0F23aFd5404b34006678c900629183c9A25d"),
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query nonce:%v\n", bal.(*mtypes.QueryResult).Data.(uint64))
	st.msgs = append(st.msgs, "QueryType_TransactionCount")

	//--------------------------------------------
	_, err = st.sender.SendSync("storage", "InitHeight", uint64(10), 1, st.from)

	if err != nil {
		fmt.Printf("******storage.InitHeight err:%v\n", err)
		return st.msgs
	}
	st.msgs = append(st.msgs, "storage.InitHeight")

	//--------------------------------------------
	bal, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_BlockNumber,
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query blockNumber:%v\n", bal.(*mtypes.QueryResult).Data.(uint64))
	st.msgs = append(st.msgs, "QueryType_BlockNumber")

	//--------------------------------------------
	bal, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Code,
		Data: &mtypes.RequestParameters{
			Address: evmCommon.HexToAddress("0x9f9E0F23aFd5404b34006678c900629183c9A25d"),
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query code:%v\n", bal.(*mtypes.QueryResult).Data.([]byte))
	st.msgs = append(st.msgs, "QueryType_Code")

	//--------------------------------------------
	bal, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Storage,
		Data: &mtypes.RequestStorage{
			Address: evmCommon.HexToAddress("0xa75Cd05BF16BbeA1759DE2A66c0472131BC5Bd8D"),
			Key:     "0x0000000000000000000000000000000000000000000000000000000000000000",
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query storage:%v\n", bal.(*mtypes.QueryResult).Data.([]byte))
	st.msgs = append(st.msgs, "QueryType_Storage")

	block, txhashes := MakeMonacoBlock()
	blockHash := evmCommon.BytesToHash(block.Blockhash)

	//------------------------
	_, err = st.sender.SendSync("blockstore", "Save", block, 10, st.from)
	if err != nil {
		fmt.Printf("******blockstore.Save err:%v\n", err)
		return st.msgs
	}
	st.msgs = append(st.msgs, "blockstore.Save")

	//--------------------------------------------
	bal, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestBlockByHeight,
		Data:      big.NewInt(10),
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query storage:%v\n", bal.(*mtypes.QueryResult).Data.(*mtypes.MonacoBlock))
	st.msgs = append(st.msgs, "QueryType_TestBlockByHeight")

	//--------------------------------------------
	bal, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestTxByPosition,
		Data: &mstypes.Position{
			Height:     10,
			IdxInBlock: 1,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query storage:%v\n", bal.(*mtypes.QueryResult).Data.(*evmTypes.Transaction))
	st.msgs = append(st.msgs, "QueryType_TestTxByPosition")

	//----------------------------------
	// txhashes := []evmCommon.Hash{
	// 	evmCommon.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
	// 	evmCommon.BytesToHash([]byte{11, 12, 13, 14, 51, 16, 17, 18}),
	// }

	keys := make([]string, len(txhashes))
	for i := range keys {
		keys[i] = string(txhashes[i].Bytes())
	}
	_, err = st.sender.SendSync("indexerstore", "Save", &storage.SaveIndexRequest{
		Height: 10,
		Keys:   keys,
		Hash:   string(blockHash.Bytes()),
		IsSave: true,
	}, 10, st.from)
	if err != nil {
		fmt.Printf("******indexerstore.Save err:%v\n", err)
		return st.msgs
	}
	st.msgs = append(st.msgs, "indexerstore.Save")

	//--------------------------------------------
	hashes, err := st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestHashesByHeight,
		Data:      big.NewInt(10),
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query storage:%v\n", hashes.(*mtypes.QueryResult).Data.([]string))
	st.msgs = append(st.msgs, "QueryType_TestHashesByHeight")

	//--------------------------------------------
	pos, err := st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestPositionByHash,
		Data:      txhashes[1],
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query pos:%v\n", pos.(*mtypes.QueryResult).Data.(*mstypes.Position))
	st.msgs = append(st.msgs, "QueryType_TestPositionByHash")

	//--------------------------------------------
	height, err := st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestHeightByHash,
		Data:      blockHash,
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query height:%v\n", height.(*mtypes.QueryResult).Data.(*big.Int))
	st.msgs = append(st.msgs, "QueryType_TestHeightByHash")

	blockHeight := big.NewInt(10)
	//--------------------
	receipts := MakeReceipts()
	_, err = st.sender.SendSync("receiptstore", "Save", &storage.SaveReceiptsRequest{
		Height:   10,
		Receipts: receipts,
	}, 3, st.from)
	if err != nil {
		fmt.Printf("******receiptstore.Save err:%v\n", err)
		return st.msgs
	}
	st.msgs = append(st.msgs, "receiptstore.Save")

	//--------------------------------------------
	rs, err := st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestReceiptsByHeight,
		Data:      blockHeight,
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query receipts:%v\n", rs.(*mtypes.QueryResult).Data.([]*evmTypes.Receipt))
	st.msgs = append(st.msgs, "QueryType_TestReceiptsByHeight")

	//--------------------------------------------
	r, err := st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestReceiptByPosition,
		Data: &mstypes.Position{
			Height:     10,
			IdxInBlock: 1,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query receipt:%v\n", r.(*mtypes.QueryResult).Data.(*evmTypes.Receipt))
	st.msgs = append(st.msgs, "QueryType_TestReceiptByPosition")

	//--------------------------------------------
	h, err := st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestHeightByHashOrNumber,
		Data: &mtypes.BlockNumberOrHash{
			// BlockNumber: big.NewInt(10),
			BlockHash: &blockHash,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query height:%v\n", h.(*mtypes.QueryResult).Data.(*big.Int))
	st.msgs = append(st.msgs, "QueryType_TestHeightByHashOrNumber")

	//--------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestHeightByHashOrNumber,
		Data: &mtypes.BlockNumberOrHash{
			BlockNumber: big.NewInt(10),
			// BlockHash: &blockHash,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query height:%v\n", h.(*mtypes.QueryResult).Data.(*big.Int))
	st.msgs = append(st.msgs, "QueryType_TestHeightByHashOrNumber")

	//--------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Block_Receipts,
		Data: &mtypes.BlockNumberOrHash{
			BlockNumber: big.NewInt(10),
			// BlockHash: &blockHash,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query Block_Receipts:%v\n", h.(*mtypes.QueryResult).Data.([]*evmTypes.Receipt))
	st.msgs = append(st.msgs, "QueryType_Block_Receipts")

	//--------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Receipt_Eth,
		Data:      txhashes[1],
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query Receipt_Eth:%v\n", h.(*mtypes.QueryResult).Data.(*evmTypes.Receipt))
	st.msgs = append(st.msgs, "QueryType_Receipt_Eth")

	//--------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TxNumsByHash,
		Data:      blockHash,
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------storage.Query TxNumsByHash:%v\n", h.(*mtypes.QueryResult).Data.(int))
	st.msgs = append(st.msgs, "QueryType_TxNumsByHash")

	//--------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TxNumsByNumber,
		Data:      blockHeight.Int64(),
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------****storage.Query TxNumsByNumber:%v\n", h.(*mtypes.QueryResult).Data.(int))
	st.msgs = append(st.msgs, "QueryType_TxNumsByNumber")

	//--------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_RawBlock,
		Data:      blockHeight.Uint64(),
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------****storage.Query RawBlock:%v\n", h.(*mtypes.QueryResult).Data.(*mtypes.MonacoBlock))
	st.msgs = append(st.msgs, "QueryType_RawBlock")

	//--------------------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestGetTransactionByPosition,
		Data: &queryplan.QueryParamTransactionByPosition{
			ChainId: big.NewInt(118),
			Position: &mstypes.Position{
				Height:     10,
				IdxInBlock: 1,
			},
			BlockHash:  blockHash,
			SignerType: 0,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	fmt.Printf("-------****storage.Query TestGetTransactionByPosition:%v\n", h.(*mtypes.QueryResult).Data.(*mtypes.RPCTransaction))
	st.msgs = append(st.msgs, "QueryType_TestGetTransactionByPosition")

	//--------------------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestRpcBlockSubPlan,
		Data: &queryplan.QueryParamRpcBlock{
			Height:     big.NewInt(10),
			Fulltx:     true,
			OnlyHeader: false,
			ChainId:    big.NewInt(118),
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	result := h.(*mtypes.QueryResult).Data.(*queryplan.RpcBlockResult)
	if result != nil {
		for i := range result.Block.Transactions {
			fmt.Printf("-------****storage.Query TestRpcBlockSubPlan transaction idx:%v     transaction:%v\n", i, result.Block.Transactions[i])
		}
	} else {
		fmt.Printf("-------****storage.Query TestRpcBlockSubPlan is nil\n")
	}

	st.msgs = append(st.msgs, "QueryType_TestRpcBlockSubPlan")

	//--------------------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TestRpcBlockSubPlan,
		Data: &queryplan.QueryParamRpcBlock{
			Height:     big.NewInt(10),
			Fulltx:     false,
			OnlyHeader: false,
			ChainId:    big.NewInt(118),
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	result = h.(*mtypes.QueryResult).Data.(*queryplan.RpcBlockResult)
	if result != nil {
		for i := range result.Block.Transactions {
			fmt.Printf("-------****storage.Query TestRpcBlockSubPlan transaction idx:%v    hash:%v\n", i, result.Block.Transactions[i])
		}

	} else {
		fmt.Printf("-------****storage.Query TestRpcBlockSubPlan is nil\n")
	}

	st.msgs = append(st.msgs, "QueryType_TestRpcBlockSubPlan")

	//--------------------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_BlocByHash,
		Data: &mtypes.RequestBlockEth{
			Hash:   blockHash,
			FullTx: true,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	rb := h.(*mtypes.QueryResult).Data.(*mtypes.RPCBlock)
	if rb != nil {
		for i := range rb.Transactions {
			fmt.Printf("-------****storage.Query TestRpcBlockSubPlan transaction idx:%v    hash:%v\n", i, rb.Transactions[i])
		}

	} else {
		fmt.Printf("-------****storage.Query TestRpcBlockSubPlan is nil\n")
	}

	st.msgs = append(st.msgs, "QueryType_BlocByHash")

	//--------------------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Block_Eth,
		Data: &mtypes.RequestBlockEth{
			Number: int64(10),
			FullTx: false,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	rb = h.(*mtypes.QueryResult).Data.(*mtypes.RPCBlock)
	if rb != nil {
		for i := range rb.Transactions {
			fmt.Printf("-------****storage.Query TestRpcBlockSubPlan transaction idx:%v    hash:%v\n", i, rb.Transactions[i])
		}
	} else {
		fmt.Printf("-------****storage.Query TestRpcBlockSubPlan is nil\n")
	}
	st.msgs = append(st.msgs, "QueryType_Block_Eth")

	//--------------------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_HeaderByHash,
		Data: &mtypes.RequestBlockEth{
			Hash: blockHash,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	rb = h.(*mtypes.QueryResult).Data.(*mtypes.RPCBlock)
	fmt.Printf("-------****storage.Query HeaderByHash  transactions:%v  rpcblock:%v \n", len(rb.Transactions), rb)
	st.msgs = append(st.msgs, "QueryType_HeaderByHash")

	//--------------------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_HeaderByNumber,
		Data: &mtypes.RequestBlockEth{
			Number: int64(10),
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	rb = h.(*mtypes.QueryResult).Data.(*mtypes.RPCBlock)
	fmt.Printf("-------****storage.Query HeaderByNumber  transactions:%v  rpcblock:%v \n", len(rb.Transactions), rb)
	st.msgs = append(st.msgs, "QueryType_HeaderByNumber")

	//--------------------------------------------------------

	// txhash := evmCommon.BytesToHash([]byte{1, 2, 3, 4, 51, 61, 17, 18})

	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_Transaction,
		Data:      txhashes[1],
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	rt := h.(*mtypes.QueryResult).Data.(*mtypes.RPCTransaction)
	fmt.Printf("-------****storage.Query Transaction :%v \n", rt)
	st.msgs = append(st.msgs, "QueryType_Transaction")

	//--------------------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TxByHashAndIdx,
		Data: &mtypes.RequestBlockEth{
			Hash:  blockHash,
			Index: 1,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	rt = h.(*mtypes.QueryResult).Data.(*mtypes.RPCTransaction)
	fmt.Printf("-------****storage.Query TxByHashAndIdx transaction:%v \n", rt)
	st.msgs = append(st.msgs, "QueryType_TxByHashAndIdx")

	//--------------------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TxByNumberAndIdx,
		Data: &mtypes.RequestBlockEth{
			Number: int64(10),
			Index:  1,
		},
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	rt = h.(*mtypes.QueryResult).Data.(*mtypes.RPCTransaction)
	fmt.Printf("-------****storage.Query TxByNumberAndIdx transaction:%v \n", rt)
	st.msgs = append(st.msgs, "QueryType_TxByNumberAndIdx")

	//--------------------------------------------------------
	h, err = st.sender.SendSync("storage", "Query", &mtypes.QueryRequest{
		QueryType: mtypes.QueryType_TxMessage,
		Data:      txhashes[1],
	}, 1, st.from)

	if err != nil {
		fmt.Printf("******storage.Query err:%v\n", err)
		return st.msgs
	}
	qmr := h.(*mtypes.QueryResult).Data.(*mtypes.QueryReplayMsgResult)
	fmt.Printf("-------****storage.Query TxMessage :%v \n", qmr)
	st.msgs = append(st.msgs, "QueryType_TxMessage")

	return st.msgs
}
