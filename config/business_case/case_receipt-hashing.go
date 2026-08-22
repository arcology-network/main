package businessCase

import (
	"time"

	"github.com/arcology-network/common-lib/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
)

type ReceiptHashingTest struct {
	basePath string

	msgs []string
}

func NewReceiptHashingTest(basePath string) *ReceiptHashingTest {
	handler := &ReceiptHashingTest{
		basePath: basePath,
		msgs:     []string{},
	}
	return handler
}

func (rht *ReceiptHashingTest) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgReceiptInfo,
		scommon.MsgSelectedReceipts,
	}, false
}

func (rht *ReceiptHashingTest) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgInclusive:      1,
		scommon.MsgBlockCompleted: 1,
		scommon.MsgReceipts:       1,
	}
}

func (rht *ReceiptHashingTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgReceiptInfo, rht.receivedReceiptInfo)
	reg.Register(scommon.MsgSelectedReceipts, rht.receivedMsgs)
}

func (rht *ReceiptHashingTest) receivedReceiptInfo(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received ReceiptInfo*******", logger.F("info", msg.Data.(*mtypes.ReceiptInfo)))

	rht.receivedMsgs(ctx)
	return nil
}

func (rht *ReceiptHashingTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	rht.msgs = append(rht.msgs, msg.Name)
	return nil
}

func (rht *ReceiptHashingTest) startTest(ss *broker.StatefulStreamer) []string {
	_, txhashes := MakeArcologyBlock()
	receipts := MakeReceipts(txhashes)

	m := scommon.NewMessageForStream(scommon.MsgReceipts, receipts)
	m.Height = 10
	ss.Send(scommon.MsgReceipts, m)
	time.Sleep(1 * time.Second)

	flags := make([]bool, len(txhashes))
	for i := range txhashes {
		flags[i] = true
	}

	inclusiveList := &types.InclusiveList{
		HashList:   txhashes,
		Successful: flags,
	}

	m = scommon.NewMessageForStream(scommon.MsgInclusive, inclusiveList)
	m.Height = 10
	ss.Send(scommon.MsgInclusive, m)
	time.Sleep(4 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgBlockCompleted, "")
	m.Height = 10
	ss.Send(scommon.MsgBlockCompleted, m)
	time.Sleep(3 * time.Second)

	return rht.msgs
}
