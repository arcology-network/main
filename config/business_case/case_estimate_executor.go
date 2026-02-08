package businessCase

import (
	"fmt"
	"math/big"
	"time"

	"github.com/arcology-network/common-lib/storage/transactional"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type EstimateExecutorTest struct {
	basePath string
	msgs     []string
	sender   actor.OutboundSender
}

func NewEstimateExecutorTest(basePath string) *EstimateExecutorTest {
	handler := &EstimateExecutorTest{
		basePath: basePath,
		msgs:     []string{},
	}
	return handler
}

func (handler *EstimateExecutorTest) Inputs() ([]string, bool) {
	return []string{}, false
}

func (handler *EstimateExecutorTest) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgApcHandle:      1,
		scommon.MsgBlockStart:     1,
		scommon.MsgParentInfo:     1,
		scommon.MsgInitialization: 1,
	}
}

func (eet *EstimateExecutorTest) SetSender(sender actor.OutboundSender) {
	eet.sender = sender
}

func (eet *EstimateExecutorTest) RegisterActions(reg actor.ActionRegistrar) {
	// reg.Register(scommon.MsgAcctHash, eet.receivedMsgs)
}

func (eet *EstimateExecutorTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	eet.msgs = append(eet.msgs, msg.Name)
	return nil
}

func (eet *EstimateExecutorTest) startTest(ss *broker.StatefulStreamer) []string {
	genesis, store, _ := MakeStateStore(eet.basePath)
	blockStart := &actor.BlockStart{
		Timestamp: big.NewInt(int64(genesis.Timestamp)),
		Coinbase:  genesis.Coinbase,
		Extra:     genesis.ExtraData,
	}
	currentinfo := &mtypes.ParentInfo{
		ParentHash:    evmCommon.BytesToHash([]byte{1, 2, 3, 4, 5, 6}),
		ParentRoot:    evmCommon.BytesToHash([]byte{7, 8, 9, 10, 11, 12}),
		ExcessBlobGas: 3500000000,
		BlobGasUsed:   20000000,
	}
	//---------------------------------------
	m := scommon.NewMessageForStream(scommon.MsgInitialization, &mtypes.Initialization{
		Store:             store,
		BlockStart:        blockStart,
		ParentInformation: currentinfo,
	})
	m.Height = 10
	ss.Send(scommon.MsgInitialization, m)
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

	//---------------------------------------
	m = scommon.NewMessageForStream(scommon.MsgApcHandle, store)
	m.Height = 10
	ss.Send(scommon.MsgApcHandle, m)
	time.Sleep(1 * time.Second)

	// response, err := eet.sender.SendSync("estimate-executor", "ExecTxs", request, 0)
	// if err != nil {
	// 	fmt.Printf("******estimate-executor.ExecTxs err:%v\n", err)
	// }

	// zoomout := 2

	// gas := response.(*core.ExecutionResult).UsedGas * zoomout
	// if gas > maxGasLimit {
	// 	gas = maxGasLimit
	// }

	resp, err := eet.sender.SendSync("transactionalstore", "AddData", &transactional.AddDataRequest{
		Data:        []byte{0, 1, 2},
		RecoverFunc: "TestByte",
	}, 10, "EstimateExecutorTest")
	if err != nil {
		fmt.Printf("******transactionalstore.AddData err:%v\n", err)
		return eet.msgs
	}
	fmt.Printf("******config/business_case/case_estimate_executor.go resp:%v\n", resp)

	return eet.msgs
}
