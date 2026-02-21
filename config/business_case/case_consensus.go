package businessCase

import (
	"time"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type ConsensusTest struct {
	basePath string
	msgs     []string
	sender   actor.OutboundSender
	dic      map[string]actor.Business
}

func NewConsensusTest(basePath string, dic map[string]actor.Business) *ConsensusTest {
	handler := &ConsensusTest{
		basePath: basePath,
		msgs:     []string{},
		dic:      dic,
	}
	return handler
}

func (ct *ConsensusTest) SetSender(sender actor.OutboundSender) {
	ct.sender = sender
}

func (ct *ConsensusTest) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgExtReapCommand,
		scommon.MsgExtTxBlocks,
		scommon.MsgExtBlockCompleted,
		scommon.MsgExtReapingList,
		scommon.MsgExtBlockStart,
		scommon.MsgConsensusMaxPeerHeight,
		scommon.MsgConsensusUp,
		scommon.MsgExtBlockEnd,
	}, false
}

func (ct *ConsensusTest) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgExtAppHash:     1,
		scommon.MsgMetaBlock:      1,
		scommon.MsgTxLocals:       1,
		scommon.MsgInitialization: 1,
	}
}

func (ct *ConsensusTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgExtReapCommand, ct.receivedMsgs)
	reg.Register(scommon.MsgExtTxBlocks, ct.receivedMsgs)
	reg.Register(scommon.MsgExtBlockCompleted, ct.receivedMsgs)
	reg.Register(scommon.MsgExtReapingList, ct.receivedMsgs)
	reg.Register(scommon.MsgExtBlockStart, ct.receivedMsgs)
	reg.Register(scommon.MsgConsensusMaxPeerHeight, ct.receivedMsgs)
	reg.Register(scommon.MsgConsensusUp, ct.receivedMsgs)
	reg.Register(scommon.MsgExtBlockEnd, ct.receivedMsgs)
}

func (ct *ConsensusTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	ct.msgs = append(ct.msgs, msg.Name)
	return nil
}

func (ct *ConsensusTest) startTest(ss *broker.StatefulStreamer) []string {
	genesis, _, _ := MakeStateStore(ct.basePath)
	blockstart := GetBlockStart(genesis)

	m := scommon.NewMessageForStream(scommon.MsgInitialization, &mtypes.Initialization{
		BlockStart: blockstart,
	})
	m.Height = 1
	ss.Send(scommon.MsgInitialization, m)
	time.Sleep(3 * time.Second)

	mb, tashes := MakeMonacoBlock()

	m = scommon.NewMessageForStream(scommon.MsgTxLocals, [][]byte{
		mb.Txs[0][1:],
		mb.Txs[1][1:],
	})
	m.Height = 1
	ss.Send(scommon.MsgTxLocals, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgMetaBlock, &mtypes.MetaBlock{
		Txs:      [][]byte{},
		Hashlist: tashes,
	})
	m.Height = 1
	ss.Send(scommon.MsgMetaBlock, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgExtAppHash, evmCommon.BytesToHash([]byte{1, 2, 33, 54, 53, 52, 51}).Bytes())
	m.Height = 2
	ss.Send(scommon.MsgExtAppHash, m)
	time.Sleep(1 * time.Second)

	return ct.msgs
}
