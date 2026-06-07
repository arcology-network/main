package businessCase

import (
	"time"

	"github.com/arcology-network/main/components/storage"
	statecommitter "github.com/arcology-network/state-engine/state/committer"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
)

type DBHandlerAsyncTest struct {
	basePath               string
	dbhandle               string
	precommitMsg           string
	generationCompletedMsg string
	commitMsg              string

	msgs []string
}

func NewDBHandlerAsyncTest(basePath, dbhandle, precommitMsg, commitMsg, generationCompletedMsg string) *DBHandlerAsyncTest {
	handler := &DBHandlerAsyncTest{
		basePath:               basePath,
		dbhandle:               dbhandle,
		precommitMsg:           precommitMsg,
		commitMsg:              commitMsg,
		generationCompletedMsg: generationCompletedMsg,
		msgs:                   []string{},
	}
	return handler
}

func (handler *DBHandlerAsyncTest) Inputs() ([]string, bool) {
	msgs := []string{scommon.MsgAcctHash}
	return msgs, false
}

func (handler *DBHandlerAsyncTest) Outputs() map[string]int {
	outputs := make(map[string]int)
	outputs[handler.dbhandle] = 1
	outputs[handler.precommitMsg] = 1
	outputs[handler.commitMsg] = 1
	outputs[handler.generationCompletedMsg] = 1
	return outputs
}

func (handler *DBHandlerAsyncTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgAcctHash, handler.receivedMsgs)
}

func (handler *DBHandlerAsyncTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	handler.msgs = append(handler.msgs, msg.Name)
	return nil
}

func (handler *DBHandlerAsyncTest) startTest(ss *broker.StatefulStreamer) []string {
	_, store, _ := MakeStateStore(handler.basePath)
	obj := &storage.InitAsyncObj{
		StateStore: store,
		Committer:  statecommitter.NewStateCommitter(store.CommittedStore(), store.GetWriters()),
	}
	m := scommon.NewMessageForStream(handler.dbhandle, obj)
	m.Height = 1
	ss.Send(handler.dbhandle, m)
	time.Sleep(1 * time.Second)
	m = scommon.NewMessageForStream(handler.precommitMsg, "")
	m.Height = 1
	ss.Send(handler.precommitMsg, m)
	time.Sleep(1 * time.Second)
	m = scommon.NewMessageForStream(handler.commitMsg, "")
	m.Height = 1
	ss.Send(handler.commitMsg, m)
	time.Sleep(1 * time.Second)
	m = scommon.NewMessageForStream(handler.generationCompletedMsg, "")
	m.Height = 1
	ss.Send(handler.generationCompletedMsg, m)
	time.Sleep(1 * time.Second)

	return handler.msgs
}
