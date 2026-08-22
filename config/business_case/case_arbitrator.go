package businessCase

import (
	"fmt"
	"time"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/scheduler/conflictor"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type ArbitratorTest struct {
	basePath string
	msgs     []string
	sender   actor.OutboundSender
}

func NewArbitratorTest(basePath string) *ArbitratorTest {
	handler := &ArbitratorTest{
		basePath: basePath,
		msgs:     []string{},
	}
	return handler
}

func (at *ArbitratorTest) Inputs() ([]string, bool) {
	return []string{}, false
}

func (at *ArbitratorTest) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgTxAccessRecords: 1,
		scommon.MsgBlockCompleted:  1,
	}
}

func (at *ArbitratorTest) SetSender(sender actor.OutboundSender) {
	at.sender = sender
}

func (at *ArbitratorTest) RegisterActions(reg actor.ActionRegistrar) {
	// reg.Register(scommon.MsgAcctHash, at.receivedMsgs)
}

func (at *ArbitratorTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	at.msgs = append(at.msgs, msg.Name)
	return nil
}

func (at *ArbitratorTest) startTest(ss *broker.StatefulStreamer) []string {
	_, _, unis := MakeStateStore(at.basePath)
	_, txhashes := MakeArcologyBlock()
	ids := make([]uint64, len(txhashes))
	for i := range txhashes {
		ids[i] = uint64(i + 1)
	}
	sendAccessRecord := MakeTxAccessRecordSet(txhashes, ids, unis)
	m := scommon.NewMessageForStream(scommon.MsgTxAccessRecords, sendAccessRecord)
	m.Height = 10
	ss.Send(scommon.MsgTxAccessRecords, m)
	time.Sleep(1 * time.Second)

	arbitrateList := make([][]evmCommon.Hash, len(txhashes))
	for i := range txhashes {
		arbitrateList[i] = []evmCommon.Hash{txhashes[i]}
	}

	resp, err := at.sender.SendSync("arbitrator", "startArbitrate", &mtypes.ArbitratorRequest{
		TxsListGroup: arbitrateList,
	}, 10, "ArbitratorTest")
	if err != nil {
		fmt.Printf("-----------ArbitratorTest test err:%v\n", err)
		return at.msgs
	}
	response := resp.(*conflictor.CollisionSummary)
	fmt.Printf("-----------ArbitratorTest result:%v\n", response)

	m = scommon.NewMessageForStream(scommon.MsgBlockCompleted, "")
	m.Height = 1
	ss.Send(scommon.MsgBlockCompleted, m)
	time.Sleep(1 * time.Second)

	return at.msgs
}
