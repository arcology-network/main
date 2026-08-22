package businessCase

import (
	"fmt"
	"time"

	"github.com/arcology-network/common-lib/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type MakeCoreTest struct {
	basePath string
	msgs     []string
	sender   actor.OutboundSender
}

func NewMakeCoreTest(basePath string) *MakeCoreTest {
	handler := &MakeCoreTest{
		basePath: basePath,
		msgs:     []string{},
	}
	return handler
}

func (mct *MakeCoreTest) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgAppHash,
		scommon.MsgParentInfo,
		scommon.MsgLocalParentInfo,
		scommon.MsgPendingBlock,
	}, false
}

func (mct *MakeCoreTest) SetSender(sender actor.OutboundSender) {
	mct.sender = sender
}

func (mct *MakeCoreTest) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgBlockStart:                 1,
		scommon.MsgSelectedTxInfo:             1,
		scommon.MsgAcctHash:                   1,
		scommon.MsgReceiptInfo:                1,
		scommon.MsgLocalParentInfo:            1,
		scommon.MsgBlockParams:                1,
		scommon.MsgWithDrawHash:               1,
		scommon.MsgSignerType:                 1,
		scommon.MsgGenerationReapingCompleted: 1,
		scommon.MsgInclusive:                  1,
	}
}

func (mct *MakeCoreTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgAppHash, mct.receivedMsgs)
	reg.Register(scommon.MsgParentInfo, mct.receivedMsgs)
	reg.Register(scommon.MsgLocalParentInfo, mct.receivedMsgs)
	reg.Register(scommon.MsgPendingBlock, mct.receivedMsgs)
}

func (mct *MakeCoreTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	mct.msgs = append(mct.msgs, msg.Name)
	return nil
}

func (mct *MakeCoreTest) startTest(ss *broker.StatefulStreamer) []string {
	genesis, _, _ := MakeStateStore(mct.basePath)
	blockstaret := GetBlockStart(genesis)

	txID := fmt.Sprintf("%d", 10)
	_, err := mct.sender.SendSync("transactionalstore", "BeginTransaction", txID, 10, "MakeCoreTest")
	if err != nil {
		fmt.Printf("******transactionalstore.BeginTransaction err:%v\n", err)
		return mct.msgs
	}
	mct.msgs = append(mct.msgs, "transactionalstore.BeginTransaction")

	m := scommon.NewMessageForStream(scommon.MsgBlockStart, blockstaret)
	m.Height = 10
	ss.Send(scommon.MsgBlockStart, m)
	time.Sleep(1 * time.Second)

	mb, tashes := MakeArcologyBlock()
	selectTxs := &mtypes.SelectedTxsInfo{
		Txs:      mb.Txs,
		HashList: tashes,
		Txhash:   txhash,
	}
	m = scommon.NewMessageForStream(scommon.MsgSelectedTxInfo, selectTxs)
	m.Height = 10
	ss.Send(scommon.MsgSelectedTxInfo, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgAcctHash, [32]byte(accthash.Bytes()))
	m.Height = 10
	ss.Send(scommon.MsgAcctHash, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgReceiptInfo, &mtypes.ReceiptInfo{
		RcptHash:  rcpthash,
		BloomInfo: evmTypes.BytesToBloom([]byte{1, 23, 45, 65, 84, 91}),
		Gasused:   uint64(232434),
	})
	m.Height = 10
	ss.Send(scommon.MsgReceiptInfo, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgLocalParentInfo, GetParentInfo())
	m.Height = 10
	ss.Send(scommon.MsgLocalParentInfo, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgBlockParams, GetBlockParams())
	m.Height = 10
	ss.Send(scommon.MsgBlockParams, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgWithDrawHash, &withdrawhash)
	m.Height = 10
	ss.Send(scommon.MsgWithDrawHash, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgSignerType, uint8(0))
	m.Height = 10
	ss.Send(scommon.MsgSignerType, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgGenerationReapingCompleted, uint8(0))
	m.Height = 10
	ss.Send(scommon.MsgGenerationReapingCompleted, m)
	time.Sleep(1 * time.Second)

	flags := make([]bool, len(tashes))
	for i := range tashes {
		flags[i] = true
	}

	inclusiveList := &types.InclusiveList{
		HashList:   tashes,
		Successful: flags,
	}

	m = scommon.NewMessageForStream(scommon.MsgInclusive, inclusiveList)
	m.Height = 10
	ss.Send(scommon.MsgInclusive, m)
	time.Sleep(1 * time.Second)

	return mct.msgs
}
