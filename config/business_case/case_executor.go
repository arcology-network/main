package businessCase

import (
	"math/big"
	"time"

	"github.com/arcology-network/common-lib/types"
	eushared "github.com/arcology-network/eu/shared"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/scheduler/workload"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/arcology-network/main/modules/exec"
)

type ExecutorTest struct {
	msgs     []string
	basepath string

	txhashes  []evmCommon.Hash
	timestamp *big.Int
}

func NewExecutorTest(basepath string) *ExecutorTest {
	handler := &ExecutorTest{
		basepath: basepath,
		msgs:     []string{},
	}
	return handler
}

func (et *ExecutorTest) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgReceipts,
		scommon.MsgEuResults,
		scommon.MsgNonceEuResults,
		scommon.MsgTxAccessRecords,
		scommon.MsgTxsExecuteResults,
		"rpcTestStart",
	}, false
}

func (et *ExecutorTest) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgInitialization:        1,
		scommon.MsgBlockStart:            1,
		scommon.MsgParentInfo:            1,
		scommon.MsgObjectCached:          1,
		scommon.MsgApcHandle:             1,
		scommon.MsgTxsToExecute:          1,
		scommon.MsgGenerationReapingList: 1,
		scommon.MsgBlockEnd:              1,
	}
}

func (et *ExecutorTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgReceipts, et.receivedReceipts)
	reg.Register(scommon.MsgEuResults, et.receivedEuResults)
	reg.Register(scommon.MsgNonceEuResults, et.receivedNonceEuResults)
	reg.Register(scommon.MsgTxAccessRecords, et.receivedTxAccessRecords)
	reg.Register(scommon.MsgTxsExecuteResults, et.receivedTxsExecuteResults)
	reg.Register("rpcTestStart", et.rpcTestStart)
	reg.Register("onExecResult", et.onExecResult)
}

func (et *ExecutorTest) receivedReceipts(ctx *actor.ActionContext) error {
	receipts := ctx.Messages[0].Data.([]*evmTypes.Receipt)
	for i := range receipts {
		ctx.ExecCtx.LogDebug("receipts", logger.F("idx", i), logger.F("receipt", receipts[i]))
	}
	et.receivedMsgs(ctx)
	return nil
}

func (et *ExecutorTest) receivedEuResults(ctx *actor.ActionContext) error {
	euresults := ctx.Messages[0].Data.(*eushared.Euresults)
	for i := range *euresults {
		ctx.ExecCtx.LogDebug("Euresult", logger.F("idx", i), logger.F("eu", (*euresults)[i]))
	}
	et.receivedMsgs(ctx)
	return nil
}

func (et *ExecutorTest) receivedNonceEuResults(ctx *actor.ActionContext) error {
	euresults := ctx.Messages[0].Data.(*eushared.Euresults)
	for i := range *euresults {
		ctx.ExecCtx.LogDebug("NonceEuresult", logger.F("idx", i), logger.F("nonceEu", (*euresults)[i]))
	}
	et.receivedMsgs(ctx)
	return nil
}

func (et *ExecutorTest) receivedTxAccessRecords(ctx *actor.ActionContext) error {
	accres := ctx.Messages[0].Data.(*eushared.TxAccessRecordSet)
	for i := range *accres {
		ctx.ExecCtx.LogDebug("TxAccessRecordSet", logger.F("idx", i), logger.F("accressRecord", (*accres)[i]))
	}
	et.receivedMsgs(ctx)
	return nil
}

func (et *ExecutorTest) receivedTxsExecuteResults(ctx *actor.ActionContext) error {
	resps := ctx.Messages[0].Data.([]*exec.ExecutorResponse)
	for i := range resps {
		ctx.ExecCtx.LogDebug("TxsExecuteResult", logger.F("idx", i), logger.F("execResult", resps[i]))
	}
	et.receivedMsgs(ctx)
	return nil
}

func (et *ExecutorTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	et.msgs = append(et.msgs, msg.Name)
	return nil
}

func (et *ExecutorTest) rpcTestStart(ctx *actor.ActionContext) error {
	//--------------------------------------
	mblock, txhashes := MakeMonacoBlock()
	stdMsgs, _ := Transfer(mblock.Txs, txhashes)
	js := workload.JobSequence{}
	for i := range stdMsgs {
		js.AddJob(stdMsgs[i])
	}
	toExecute := &mtypes.ExecutorRequest{
		Timestamp:     et.timestamp,
		GenerationIdx: 0,
		ExecId:        0,
		Height:        10,
		JobSequences:  []*workload.JobSequence{&js},
	}
	et.txhashes = txhashes
	// ctx.ExecCtx.Send(scommon.MsgTxsToExecute, toExecute, 10)
	ctx.ExecCtx.InvokeRPC("executor", "startExecute", toExecute, "onExecResult")
	// time.Sleep(1 * time.Second)
	return nil
}

func (et *ExecutorTest) onExecResult(ctx *actor.ActionContext) error {
	//--------------------------------------------------
	flags := make([]bool, len(et.txhashes))
	for i := range et.txhashes {
		flags[i] = true
	}
	list := &types.InclusiveList{
		HashList:          et.txhashes,
		Successful:        flags,
		NextGenerationIdx: 0,
	}
	ctx.ExecCtx.Send(scommon.MsgGenerationReapingList, list, 10)
	// time.Sleep(1 * time.Second)
	//--------------------------------------------------
	ctx.ExecCtx.Send(scommon.MsgBlockEnd, "", 10)
	// time.Sleep(1 * time.Second)

	return nil
}

func (et *ExecutorTest) startTestRpc(ss *broker.StatefulStreamer) []string {
	//--------------------
	genesis, store, _ := MakeStateStore(et.basepath)
	blockStart := GetBlockStart(genesis)

	et.timestamp = blockStart.Timestamp

	currentinfo := GetParentInfo()
	m := scommon.NewMessageForStream(scommon.MsgInitialization, &mtypes.Initialization{
		Store:             store,
		BlockStart:        blockStart,
		ParentInformation: currentinfo,
	})
	m.Height = 9
	ss.Send(scommon.MsgInitialization, m)
	time.Sleep(1 * time.Second)
	//---------------------------
	list := &types.InclusiveList{
		HashList:          []evmCommon.Hash{},
		Successful:        []bool{},
		NextGenerationIdx: 0,
	}
	m = scommon.NewMessageForStream(scommon.MsgGenerationReapingList, list)
	m.Height = 9
	ss.Send(scommon.MsgGenerationReapingList, m)
	time.Sleep(1 * time.Second)

	//--------------------------------------------------
	m = scommon.NewMessageForStream(scommon.MsgBlockEnd, "")
	m.Height = 9
	ss.Send(scommon.MsgBlockEnd, m)
	time.Sleep(1 * time.Second)
	//-----------------------------------
	m = scommon.NewMessageForStream(scommon.MsgBlockStart, blockStart)
	m.Height = 10
	ss.Send(scommon.MsgBlockStart, m)
	time.Sleep(1 * time.Second)
	//------------------------------------------

	m = scommon.NewMessageForStream(scommon.MsgParentInfo, currentinfo)
	m.Height = 10
	ss.Send(scommon.MsgParentInfo, m)
	time.Sleep(1 * time.Second)
	//--------------------------------------------------
	m = scommon.NewMessageForStream(scommon.MsgObjectCached, "")
	m.Height = 10
	ss.Send(scommon.MsgObjectCached, m)
	time.Sleep(1 * time.Second)
	//----------------------------------------------------
	m = scommon.NewMessageForStream(scommon.MsgApcHandle, store)
	m.Height = 10
	ss.Send(scommon.MsgApcHandle, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream("rpcTestStart", "")
	m.Height = 10
	ss.Send("rpcTestStart", m)
	time.Sleep(3 * time.Second)

	return et.msgs
}
