package businessCase

import (
	"fmt"
	"time"

	"github.com/arcology-network/common-lib/crdt/statecell"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/scheduler/conflictor"

	statecache "github.com/arcology-network/state-engine/state/cache"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

// ---------------------------------------------------
type MockArbitrator struct {
	basePath string
	msgs     []string
}

func NewMockArbitrator(basePath string) *MockArbitrator {
	schd := &MockArbitrator{
		basePath: basePath,
		msgs:     []string{},
	}
	return schd
}

func (me *MockArbitrator) Inputs() ([]string, bool) {
	return []string{
		// scommon.MsgExecGeneration,
	}, false
}

func (ma *MockArbitrator) Outputs() map[string]int {
	return map[string]int{
		"generationEnd": 1,
	}
}

func (ma *MockArbitrator) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("startArbitrate", ma.startArbitrate)
}

func (ma *MockArbitrator) RpcConfig() (string, int) {
	return "arbitrator", 20
}

func (ma *MockArbitrator) startArbitrate(ctx *actor.ActionContext) error {
	ctx.ExecCtx.LogDebug("startRpc", logger.F("data", ctx.Messages[0].Data), logger.F("reqID", ctx.Messages[0].ReqID))

	resp := &conflictor.CollisionSummary{}

	ctx.ExecCtx.SendRpcResponse("", resp)
	ctx.ExecCtx.Send("generationEnd", "")
	return nil
}

// ----------------------------------------
type MockExecutor struct {
	basePath string
	msgs     []string
}

func NewMockExecutor(basePath string) *MockExecutor {
	schd := &MockExecutor{
		basePath: basePath,
		msgs:     []string{},
	}
	return schd
}

func (me *MockExecutor) Inputs() ([]string, bool) {
	return []string{
		// scommon.MsgExecGeneration,
	}, false
}

func (me *MockExecutor) Outputs() map[string]int {
	return map[string]int{}
}

func (me *MockExecutor) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("startExecute", me.startExecute)
}

func (me *MockExecutor) RpcConfig() (string, int) {
	return "executor", 20
}

func (me *MockExecutor) startExecute(ctx *actor.ActionContext) error {
	ctx.ExecCtx.LogDebug("startExecute", logger.F("data", ctx.Messages[0].Data), logger.F("reqID", ctx.Messages[0].ReqID))

	res := []*mtypes.JobSequenceResponse{}
	params := ctx.RPC.Request.(*mtypes.ExecutorRequest)
	for _, sequence := range params.JobSequences {
		resp := make([]*mtypes.ExecuteResponse, 0, len(sequence.Jobs))
		for k := range sequence.Jobs {
			resp = append(resp, &mtypes.ExecuteResponse{
				Hash:    sequence.Jobs[k].StdMsg.TxHash,
				Status:  1,
				GasUsed: 2000000,
			})
		}
		res = append(res, &mtypes.JobSequenceResponse{
			Responses:       resp,
			ContractAddress: []evmCommon.Address{},
			CallResults:     [][]byte{},
		})
	}

	ctx.ExecCtx.SendRpcResponse("", &mtypes.ExecResponses{
		Resp:   res,
		ExecId: 0,
	})
	return nil
}

// -----------------------------------------
type SchedulerTest struct {
	basePath string
	msgs     []string
	store    *statecache.ExecutionStateStore
	unis     []*statecell.StateCell

	sender actor.OutboundSender
}

func NewSchedulerTest(basePath string) *SchedulerTest {
	schd := &SchedulerTest{
		basePath: basePath,
		msgs:     []string{},
	}
	return schd
}

func (st *SchedulerTest) SetSender(sender actor.OutboundSender) {
	st.sender = sender
}

func (st *SchedulerTest) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgInclusive,
		// scommon.MsgSchdState,
		scommon.MsgGenerationReapingList,
		scommon.MsgGenerationReapingCompleted,
		// scommon.MsgExecGeneration,
		"generationEnd",
	}, false
}

func (st *SchedulerTest) Outputs() map[string]int {
	return map[string]int{
		// scommon.MsgInitScheduletate: 1,
		actor.CombinedName(scommon.MsgMessagersReaped, scommon.MsgBlockStart): 1,
		scommon.MsgFeedBacks:      1,
		scommon.MsgExecGeneration: 1,
		scommon.MsgApcHandle:      1,
	}
}

func (st *SchedulerTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgInclusive, st.receivedMsgs)
	// reg.Register(scommon.MsgSchdState, st.receivedMsgs)
	reg.Register(scommon.MsgGenerationReapingList, st.receivedMsgs)
	reg.Register(scommon.MsgGenerationReapingCompleted, st.receivedMsgs)
	reg.Register("generationEnd", st.receivedGenerationEnd)
}

func (st *SchedulerTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	st.msgs = append(st.msgs, msg.Name)
	return nil
}

func (st *SchedulerTest) receivedGenerationEnd(ctx *actor.ActionContext) error {
	//-------------------------------------------------------
	ctx.ExecCtx.Send(scommon.MsgApcHandle, st.store, 10)
	time.Sleep(1 * time.Second)

	//-------------------------
	ctx.ExecCtx.Send(scommon.MsgFeedBacks, st.unis, 10)
	time.Sleep(1 * time.Second)

	st.receivedMsgs(ctx)
	return nil
}

func (st *SchedulerTest) startTest(ss *broker.StatefulStreamer) []string {
	genesis, store, univals := MakeStateStore(st.basePath)
	st.store = store
	st.unis = univals
	blockStart := GetBlockStart(genesis)
	mblock, txhashes := MakeArcologyBlock()
	msgs, _ := Transfer(mblock.Txs, txhashes)

	//------------------------
	// var na int
	txID := fmt.Sprintf("%d", 10)
	_, err := st.sender.SendSync("transactionalstore", "BeginTransaction", txID, 1, "transactionalstore")
	if err != nil {
		fmt.Printf("******transactionalstore.BeginTransaction err:%v\n", err)
		return st.msgs
	}
	st.msgs = append(st.msgs, "transactionalstore.BeginTransaction")

	//-------------------
	m := scommon.NewMessageForStream(scommon.MsgInitialization, &mtypes.Initialization{
		Store: store,
	})
	m.Height = 10
	ss.Send(scommon.MsgInitialization, m)
	time.Sleep(1 * time.Second)

	//--------------------------
	m = scommon.NewMessageForStream(scommon.MsgBlockStart, blockStart)
	m.Height = 10
	ss.Send(scommon.MsgBlockStart, m)
	time.Sleep(1 * time.Second)

	//-------------------------------
	m = scommon.NewMessageForStream(scommon.MsgMessagersReaped, msgs)
	m.Height = 10
	ss.Send(scommon.MsgMessagersReaped, m)
	time.Sleep(5 * time.Second)

	return st.msgs
}
