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
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type GatewayTppTest struct {
	basePath string
	msgs     []string
	sender   actor.OutboundSender
}

func NewGatewayTppTest(basePath string) *GatewayTppTest {
	handler := &GatewayTppTest{
		basePath: basePath,
		msgs:     []string{},
	}
	return handler
}

func (gtt *GatewayTppTest) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgTxLocalsUnChecked,
		scommon.MsgCheckedTxs,
		scommon.MsgTxLocals,
		scommon.MsgCheckingTxs,
		scommon.MsgMessager,
		scommon.MsgRpcHash,
	}, false
}

func (gtt *GatewayTppTest) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgTxBlocks:   1,
		scommon.MsgSignerType: 1,
	}
}

func (gtt *GatewayTppTest) SetSender(sender actor.OutboundSender) {
	gtt.sender = sender
}

func (gtt *GatewayTppTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgTxLocalsUnChecked, gtt.receivedMsgs)
	reg.Register(scommon.MsgCheckedTxs, gtt.receivedMsgs)
	reg.Register(scommon.MsgTxLocals, gtt.receivedMsgs)
	reg.Register(scommon.MsgCheckingTxs, gtt.receivedMsgs)
	reg.Register(scommon.MsgMessager, gtt.receivedMessager)
	reg.Register(scommon.MsgRpcHash, gtt.receivedMsgs)
}

func (gtt *GatewayTppTest) receivedMessager(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	msgs := msg.Data.(*types.StdTransactionPack)
	ctx.ExecCtx.LogDebug("Received Messager *******", logger.F("messages", len(msgs.Txs)))

	gtt.receivedMsgs(ctx)
	return nil
}

func (gtt *GatewayTppTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	gtt.msgs = append(gtt.msgs, msg.Name)
	return nil
}

func (gtt *GatewayTppTest) startTest(ss *broker.StatefulStreamer) []string {
	//-------------------------------------------
	m := scommon.NewMessageForStream(scommon.MsgSignerType, uint8(0))
	m.Height = 1
	ss.Send(scommon.MsgSignerType, m)
	time.Sleep(1 * time.Second)

	mblock, _ := MakeArcologyBlock()
	txblocks := &types.IncomingTxs{
		Txs:       mblock.Txs,
		Src:       types.NewTxSource(types.TxSourceConsensus, "proposer"),
		RequestID: "",
	}
	m = scommon.NewMessageForStream(scommon.MsgTxBlocks, txblocks)
	m.Height = 1
	ss.Send(scommon.MsgTxBlocks, m)
	time.Sleep(1 * time.Second)

	return gtt.msgs
}

func (gtt *GatewayTppTest) startTestLocalSingle(ss *broker.StatefulStreamer) []string {
	gtt.msgs = []string{}
	//-------------------------------------------
	m := scommon.NewMessageForStream(scommon.MsgSignerType, uint8(0))
	m.Height = 1
	ss.Send(scommon.MsgSignerType, m)
	time.Sleep(1 * time.Second)

	mblock, _ := MakeArcologyBlock()
	response, err := gtt.sender.SendSync("gateway", "SendRawTransaction", &mtypes.RawTransactionArgs{
		Tx: mblock.Txs[0][1:],
	}, 0)
	if err != nil {
		fmt.Printf("*********config/business_case/case_gateway_tpp.go---err:%v\n", err)
		return []string{}
	}
	fmt.Printf("*********config/business_case/case_gateway_tpp.go---response:%v\n", response.(*mtypes.RawTransactionReply).TxHash.(evmCommon.Hash))

	return gtt.msgs
}

func (gtt *GatewayTppTest) startTestLocalBatch(ss *broker.StatefulStreamer) []string {
	gtt.msgs = []string{}
	//-------------------------------------------
	m := scommon.NewMessageForStream(scommon.MsgSignerType, uint8(0))
	m.Height = 1
	ss.Send(scommon.MsgSignerType, m)
	time.Sleep(1 * time.Second)

	mblock, _ := MakeArcologyBlock()
	rawTxs := make([][]byte, len(mblock.Txs))
	for i := range rawTxs {
		rawTxs[i] = mblock.Txs[i][1:]
	}

	_, err := gtt.sender.SendSync("gateway", "ReceivedTransactions", &mtypes.SendTransactionArgs{
		Txs: rawTxs,
	}, 0)
	if err != nil {
		fmt.Printf("*********config/business_case/case_gateway_tpp.go---err:%v\n", err)
		return []string{}
	}

	time.Sleep(3 * time.Second)

	return gtt.msgs
}
