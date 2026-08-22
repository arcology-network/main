package businessCase

import (
	"log"
	"time"

	"github.com/arcology-network/common-lib/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type PoolTest struct {
	basePath string
	sender   actor.OutboundSender
	msgs     []string

	list    []evmCommon.Hash
	runAsL1 bool
}

func NewPoolTest(basePath string, runAsL1 bool) *PoolTest {
	handler := &PoolTest{
		basePath: basePath,
		runAsL1:  runAsL1,
		list:     []evmCommon.Hash{},
		msgs:     []string{},
	}
	return handler
}

func (handler *PoolTest) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgMessagersReaped,
		scommon.MsgMetaBlock,
		scommon.MsgSelectedTxInfo,
		scommon.MsgBlockParams,
		scommon.MsgWithDrawHash,
		scommon.MsgSignerType,
		scommon.MsgOpCommand,
		// "opRequest",
	}, false
}

func (handler *PoolTest) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgNonceReady:       1,
		scommon.MsgMessager:         1,
		scommon.MsgReapCommand:      1,
		scommon.MsgReapinglist:      1,
		scommon.MsgSelectedReceipts: 1,
		scommon.MsgPendingBlock:     1,
		scommon.MsgInitialization:   1,
		// "opRequest":                 1,
	}
}

func (pt *PoolTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgMessagersReaped, pt.receivedMessagersReaped)
	reg.Register(scommon.MsgMetaBlock, pt.receivedMsgs)
	reg.Register(scommon.MsgSelectedTxInfo, pt.receivedMsgs)
	reg.Register(scommon.MsgBlockParams, pt.receivedMsgs)
	reg.Register(scommon.MsgWithDrawHash, pt.receivedMsgs)
	reg.Register(scommon.MsgSignerType, pt.receivedMsgs)
	reg.Register(scommon.MsgOpCommand, pt.receivedMsgs)
	// reg.Register("opRequest", pt.opRequest)
	// reg.Register("opRequestResult", pt.opRequestResult)
}

func (pt *PoolTest) SetSender(sender actor.OutboundSender) {
	pt.sender = sender
}

func (pt *PoolTest) receivedMessagersReaped(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	stdmsgs := msg.Data.([]*types.StandardMessage)
	ctx.ExecCtx.LogDebug("Reaped Messages *******", logger.F("count", len(stdmsgs)))

	if !pt.runAsL1 {
		dic := map[evmCommon.Hash]int{}
		for i := range pt.list {
			dic[pt.list[i]] = 1
		}
		for i := range stdmsgs {
			ctx.ExecCtx.LogErr("Reaped Message *******", logger.F("hash", stdmsgs[i].TxHash))
			if _, ok := dic[stdmsgs[i].TxHash]; !ok {
				ctx.ExecCtx.LogErr("Reaped Message not in list======", logger.F("hash", stdmsgs[i].TxHash))
			}
		}
	}

	pt.receivedMsgs(ctx)
	return nil
}

func (pt *PoolTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	pt.msgs = append(pt.msgs, msg.Name)
	return nil
}

func (pt *PoolTest) startTestAsL1(ss *broker.StatefulStreamer) []string {
	genesis, store, _ := MakeStateStore(pt.basePath)

	//-----------------------------------
	m := scommon.NewMessageForStream(scommon.MsgInitialization, &mtypes.Initialization{
		Store:       store,
		ChainConfig: genesis.Config,
	})
	m.Height = 10
	ss.Send(scommon.MsgInitialization, m)
	time.Sleep(1 * time.Second)

	//-------------------------
	mblock, txhashes := MakeArcologyBlock()
	_, stdTxs := Transfer(mblock.Txs, txhashes)

	pack := &types.StdTransactionPack{
		Txs: stdTxs,
		Src: types.NewTxSource(types.TxSourceLocal, "ethapi"),
	}

	m = scommon.NewMessageForStream(scommon.MsgMessager, pack)
	m.Height = 10
	ss.Send(scommon.MsgMessager, m)
	time.Sleep(1 * time.Second)

	//-----------------------------------------------------------
	m = scommon.NewMessageForStream(scommon.MsgReapCommand, "")
	m.Height = 10
	ss.Send(scommon.MsgReapCommand, m)
	time.Sleep(1 * time.Second)

	//----------------------------------------------

	reaplist := &types.ReapingList{
		List: txhashes,
	}
	m = scommon.NewMessageForStream(scommon.MsgReapinglist, reaplist)
	m.Height = 1
	ss.Send(scommon.MsgReapinglist, m)
	time.Sleep(1 * time.Second)

	//----------------------------------------------------------------------
	receipts := MakeReceipts(txhashes)
	m = scommon.NewMessageForStream(scommon.MsgSelectedReceipts, receipts)
	m.Height = 1
	ss.Send(scommon.MsgSelectedReceipts, m)
	time.Sleep(1 * time.Second)

	//----------------------------------------------------------------------

	m = scommon.NewMessageForStream(scommon.MsgPendingBlock, BlockWithHeader(mblock, genesis))
	m.Height = 1
	ss.Send(scommon.MsgPendingBlock, m)
	time.Sleep(1 * time.Second)

	return pt.msgs
}

func (pt *PoolTest) startTestAsL2(ss *broker.StatefulStreamer) []string {
	genesis, store, _ := MakeStateStore(pt.basePath)

	//-----------------------------------
	m := scommon.NewMessageForStream(scommon.MsgInitialization, &mtypes.Initialization{
		Store:       store,
		ChainConfig: genesis.Config,
	})
	m.Height = 10
	ss.Send(scommon.MsgInitialization, m)
	time.Sleep(1 * time.Second)

	//-------------------------
	mblock, txhashes := MakeArcologyBlock()
	_, stdTxs := Transfer(mblock.Txs, txhashes)

	pack := &types.StdTransactionPack{
		Txs: stdTxs,
		Src: types.NewTxSource(types.TxSourceLocal, "ethapi"),
	}

	m = scommon.NewMessageForStream(scommon.MsgMessager, pack)
	m.Height = 10
	ss.Send(scommon.MsgMessager, m)
	time.Sleep(1 * time.Second)

	//-----------------------------------------------------------
	m = scommon.NewMessageForStream(scommon.MsgReapCommand, "")
	m.Height = 10
	ss.Send(scommon.MsgReapCommand, m)
	time.Sleep(1 * time.Second)

	//----------------------------------------------
	reaplist := &types.ReapingList{
		List: txhashes,
	}
	m = scommon.NewMessageForStream(scommon.MsgReapinglist, reaplist)
	m.Height = 10
	ss.Send(scommon.MsgReapinglist, m)
	time.Sleep(1 * time.Second)

	//----------------------------------------------------------------------
	receipts := MakeReceipts(txhashes)
	m = scommon.NewMessageForStream(scommon.MsgSelectedReceipts, receipts)
	m.Height = 10
	ss.Send(scommon.MsgSelectedReceipts, m)
	time.Sleep(1 * time.Second)

	//----------------------------------------------------------------------

	m = scommon.NewMessageForStream(scommon.MsgPendingBlock, BlockWithHeader(mblock, genesis))
	m.Height = 10
	ss.Send(scommon.MsgPendingBlock, m)
	time.Sleep(2 * time.Second)

	// //-----------------------------------------------------------
	// m = scommon.NewMessageForStream("opRequest", "")
	// m.Height = 10
	// ss.Send("opRequest", m)
	// time.Sleep(2 * time.Second)
	//-----------------------------------------------------------------
	// msg := ctx.Messages[0]
	tx1 := evmCommon.Hex2Bytes("0002f8667601010382c24294b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f8084a523b88ac080a0a895c4bffe025acf794a45f99ad17d5d99fb1b12f20db24074badfb9643bb3f6a0207568ef208ca23d748104af2557e55809d0a18604b3f3b552a7c0429265adfb")
	txhash1 := evmCommon.BytesToHash([]byte{98, 67, 45, 32, 22, 33, 55, 66, 77, 88})
	_, stdTxs1 := Transfer([][]byte{tx1}, []evmCommon.Hash{txhash1})

	pt.list = []evmCommon.Hash{txhash1, txhashes[1]}

	ret, err := pt.sender.SendSync("pool", "ReceivedMessages", &mtypes.OpRequest{
		BlockParam:   GetBlockParams(),
		Withdrawals:  evmTypes.Withdrawals{},
		Transactions: stdTxs1,
	}, 10, "pooltest")
	if err != nil {
		log.Printf("call pool.ReceivedMessages err:%v", err)
		return pt.msgs
	}
	pt.msgs = append(pt.msgs, "pool.ReceivedMessages")
	time.Sleep(2 * time.Second)

	result := ret.(*mtypes.BlockResult)
	log.Printf("call pool.ReceivedMessages result:%v", result)
	return pt.msgs
}
