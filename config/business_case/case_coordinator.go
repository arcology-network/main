package businessCase

import (
	"time"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
)

type CoordinatorTest struct {
	basePath string
	msgs     []string
}

func NewCoordinatorTest(basePath string) *CoordinatorTest {
	handler := &CoordinatorTest{
		basePath: basePath,
		msgs:     []string{},
	}
	return handler
}

func (handler *CoordinatorTest) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgBlockStart,
		scommon.MsgBlockEnd,
		scommon.MsgReapCommand,
		scommon.MsgTxBlocks,
		scommon.MsgBlockCompleted,
		scommon.MsgReapinglist,
		scommon.MsgMetaBlock,
		scommon.MsgExtAppHash,
		// scommon.MsgStateSyncStart,
	}, false
}

func (handler *CoordinatorTest) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgInitialization:         1,
		scommon.MsgConsensusMaxPeerHeight: 1,
		scommon.MsgConsensusUp:            1,
		scommon.MsgExtBlockStart:          1,
		scommon.MsgExtBlockEnd:            1,
		scommon.MsgExtReapCommand:         1,
		scommon.MsgExtTxBlocks:            1,
		scommon.MsgExtBlockCompleted:      1,
		scommon.MsgExtReapingList:         1,
		scommon.MsgAppHash:                1,
		scommon.MsgStateSyncDone:          1,
	}
}

func (ct *CoordinatorTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgBlockStart, ct.receivedMsgs)
	reg.Register(scommon.MsgBlockEnd, ct.receivedMsgs)
	reg.Register(scommon.MsgReapCommand, ct.receivedMsgs)
	reg.Register(scommon.MsgTxBlocks, ct.receivedMsgs)
	reg.Register(scommon.MsgBlockCompleted, ct.receivedMsgs)
	reg.Register(scommon.MsgReapinglist, ct.receivedMsgs)
	reg.Register(scommon.MsgMetaBlock, ct.receivedMsgs)
	reg.Register(scommon.MsgExtAppHash, ct.receivedMsgs)
	reg.Register(scommon.MsgStateSyncStart, ct.receivedMsgs)
}

func (ct *CoordinatorTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	ct.msgs = append(ct.msgs, msg.Name)
	return nil
}

func (ct *CoordinatorTest) startTest(ss *broker.StatefulStreamer) []string {
	genesis, _, _ := MakeStateStore(ct.basePath)
	blockstart := GetBlockStart(genesis)

	m := scommon.NewMessageForStream(scommon.MsgInitialization, &mtypes.Initialization{
		BlockStart: blockstart,
	})
	m.Height = 1
	ss.Send(scommon.MsgInitialization, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgConsensusMaxPeerHeight, uint64(10))
	m.Height = 1
	ss.Send(scommon.MsgConsensusMaxPeerHeight, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgConsensusUp, "")
	m.Height = 1
	ss.Send(scommon.MsgConsensusUp, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgExtBlockStart, blockstart)
	m.Height = 1
	ss.Send(scommon.MsgExtBlockStart, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgExtBlockEnd, "")
	m.Height = 1
	ss.Send(scommon.MsgExtBlockEnd, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgExtReapCommand, "")
	m.Height = 1
	ss.Send(scommon.MsgExtReapCommand, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgExtTxBlocks, "")
	m.Height = 1
	ss.Send(scommon.MsgExtTxBlocks, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgExtBlockCompleted, "")
	m.Height = 1
	ss.Send(scommon.MsgExtBlockCompleted, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgExtReapingList, "")
	m.Height = 1
	ss.Send(scommon.MsgExtReapingList, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgAppHash, "")
	m.Height = 1
	ss.Send(scommon.MsgAppHash, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgStateSyncDone, "")
	m.Height = 1
	ss.Send(scommon.MsgStateSyncDone, m)
	time.Sleep(1 * time.Second)

	return ct.msgs
}
