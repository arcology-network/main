package businessCase

import (
	"fmt"
	"math/big"
	"time"

	"github.com/arcology-network/main/modules/exec"
	mtypes "github.com/arcology-network/main/types"
	statestore "github.com/arcology-network/storage-committer"
	univaluepk "github.com/arcology-network/storage-committer/type/univalue"
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

	ctx.ExecCtx.SendRpcResponse("", &mtypes.ArbitratorResponse{
		CPairLeft:  []uint64{},
		CPairRight: []uint64{},
	})
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

	res := []*mtypes.ExecuteResponse{}
	params := ctx.RPC.Request.(*mtypes.ExecutorRequest)
	total := 0
	for _, sequence := range params.Sequences {
		total = total + len(sequence.Msgs)

		for k := range sequence.Msgs {
			res = append(res, &mtypes.ExecuteResponse{
				Hash:    sequence.Msgs[k].TxHash,
				Status:  1,
				GasUsed: 2000000,
			})
		}
	}
	totalGroups := total
	// ctx.ExecCtx.Send(scommon.MsgTxsToExecute, params, params.Height)
	//----------------------------------------------------
	resp := []*exec.ExecutorResponse{
		&exec.ExecutorResponse{
			Responses:       res,
			ContractAddress: []evmCommon.Address{},
			CallResults:     [][]byte{},
		},
	}
	//----------------------------------------------
	resultLength := 0

	HashList := make([]evmCommon.Hash, 0, totalGroups)
	StatusList := make([]uint64, 0, totalGroups)
	GasUsedList := make([]uint64, 0, totalGroups)
	contractAddress := []evmCommon.Address{}

	callResults := make([][]byte, 0, totalGroups)

	for _, exectorResponse := range resp {

		contractAddress = append(contractAddress, exectorResponse.ContractAddress...)

		resultLength = resultLength + len(exectorResponse.Responses)
		for _, txResponse := range exectorResponse.Responses {

			HashList = append(HashList, txResponse.Hash)
			StatusList = append(StatusList, txResponse.Status)
			GasUsedList = append(GasUsedList, txResponse.GasUsed)
		}

		callResults = append(callResults, exectorResponse.CallResults...)
	}

	ctx.ExecCtx.LogDebug("Exec return results", logger.F("txResults", resultLength))

	ctx.ExecCtx.SendRpcResponse("", &mtypes.ExecutorResponses{
		HashList:          HashList,
		StatusList:        StatusList,
		GasUsedList:       GasUsedList,
		ContractAddresses: contractAddress,
		CallResults:       callResults,
	})
	return nil
}

// -----------------------------------------
type SchedulerTest struct {
	basePath string
	msgs     []string
	store    *statestore.StateStore
	unis     []*univaluepk.Univalue

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
		scommon.MsgSchdState,
		scommon.MsgGenerationReapingList,
		scommon.MsgGenerationReapingCompleted,
		// scommon.MsgExecGeneration,
		"generationEnd",
	}, false
}

func (st *SchedulerTest) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgInitScheduletate: 1,
		actor.CombinedName(scommon.MsgMessagersReaped, scommon.MsgBlockStart): 1,
		scommon.MsgFeedBacks:      1,
		scommon.MsgExecGeneration: 1,
		scommon.MsgApcHandle:      1,
	}
}

func (st *SchedulerTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgInclusive, st.receivedMsgs)
	reg.Register(scommon.MsgSchdState, st.receivedMsgs)
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
	blockStart := &actor.BlockStart{
		Timestamp: big.NewInt(int64(genesis.Timestamp)),
		Coinbase:  genesis.Coinbase,
		Extra:     genesis.ExtraData,
	}
	mblock, txhashes := MakeMonacoBlock()
	msgs := Transfer(mblock, txhashes)

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
	m := scommon.NewMessageForStream(scommon.MsgInitScheduletate, []mtypes.SchdState{})
	m.Height = 10
	ss.Send(scommon.MsgInitScheduletate, m)
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
