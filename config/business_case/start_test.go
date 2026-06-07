/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package businessCase

import (
	"fmt"
	"testing"

	consensuspk "github.com/arcology-network/main/modules/consensus"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/actor/rpc"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/spf13/viper"
)

func TestConsensus(t *testing.T) {
	basepath := "./case_consensus"
	ClearPath(basepath)

	viper.Set("home", basepath)
	consensuspk.InitCfg()

	app, broker, dic := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_consensus.yaml")

	testc := NewConsensusTest(basepath, dic)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTest(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 5 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
}

// func TestHandleAsyncGeneral(t *testing.T) {
// 	basepath := "./handleAsyncGeneral"
// 	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_handler_async_general.yaml")

// 	testc := NewDBHandlerAsyncTest(basepath, scommon.MsgGeneralDB, scommon.MsgGeneralPrecommit, scommon.MsgGeneralCommit, scommon.MsgGeneralCompleted)
// 	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})

// 	StartSys(app, broker)

// 	msgs := testc.startTest(broker)
// 	fmt.Printf("Received Msg list:%v\n", msgs)
// 	if len(msgs) != 1 {
// 		t.Errorf("test err,msg counter:%v", len(msgs))
// 	}

// 	ClearPath(basepath)
// }

// func TestHandleAsyncNonce(t *testing.T) {
// 	basepath := "./handleAsyncNonce"
// 	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_handler_async_nonce.yaml")

// 	testc := NewDBHandlerAsyncTest(basepath, scommon.MsgNonceDB, scommon.MsgNoncePrecommit, scommon.MsgNonceCommit, scommon.MsgNonceCompleted)
// 	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})

// 	StartSys(app, broker)

// 	msgs := testc.startTest(broker)
// 	fmt.Printf("Received Msg list:%v\n", msgs)
// 	if len(msgs) != 0 {
// 		t.Errorf("test err,msg counter:%v", len(msgs))
// 	}
// 	ClearPath(basepath)
// }

func TestUrlAggrGeneral(t *testing.T) {
	basepath := "./urlAggrGeneral"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_url_aggr_general.yaml")

	testc := NewUrlAggrSelector(basepath, scommon.MsgEuResults, scommon.MsgExecuted, scommon.MsgGenerationReapingCompleted, scommon.MsgBlockEnd,
		scommon.MsgApcHandle, scommon.MsgGeneralDB, scommon.MsgGeneralCompleted, scommon.MsgGeneralPrecommit, scommon.MsgGeneralCommit,
		scommon.MsgEuResults, scommon.MsgGenerationReapingList, scommon.MsgBlockEnd, "executed")
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})

	StartSys(app, broker)

	//outDBMsg   outPrecommitMsg  MsgUrlUpdate  apcHandleName(generation) outGenerationCompletedMsg  outCommitMsg  MsgObjectCached SelectEuresult
	msgs := testc.startTest(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 3 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestUrlAggrNonce(t *testing.T) {
	basepath := "./urlAggrNonce"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_url_aggr_nonce.yaml")

	testc := NewUrlAggrSelector(basepath, scommon.MsgNonceEuResults, scommon.MsgCommitNonceUrl, scommon.MsgGenerationReapingCompleted, scommon.MsgBlockEnd,
		scommon.MsgNonceReady, scommon.MsgNonceDB, scommon.MsgNonceCompleted, scommon.MsgNoncePrecommit, scommon.MsgNonceCommit,
		scommon.MsgNonceEuResults, scommon.MsgGenerationReapingList, scommon.MsgBlockEnd, "commitNonceUrl")
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})

	StartSys(app, broker)
	//outDBMsg   outPrecommitMsg  outGenerationCompletedMsg  outCommitMsg  apcHandleName(block) SelectEuresult
	msgs := testc.startTest(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 2 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestStorageStore(t *testing.T) {
	basepath := "./case_storage_store"

	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_storage_store.yaml")

	testc := NewStorageStore(basepath)
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})

	StartSys(app, broker)

	msgs := testc.startTestTransactional(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 4 {
		t.Errorf("test Transactional store err,msg counter:%v", len(msgs))
	}

	// msgs = testc.startTestUrlStore(broker)
	// fmt.Printf("Received Msg list:%v\n", msgs)
	// if len(msgs) != 5 {
	// 	t.Errorf("test Url store err,msg counter:%v", len(msgs))
	// }

	msgs = testc.startTestTmBlockstore(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 14 {
		t.Errorf("test Tm block store err,msg counter:%v", len(msgs))
	}

	msgs = testc.startTestTmStatestore(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 9 {
		t.Errorf("test Tm state store err,msg counter:%v", len(msgs))
	}

	msgs = testc.startTestStatestore(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 3 {
		t.Errorf("test tate store err,msg counter:%v", len(msgs))
	}

	// msgs = testc.startTestSchdStore(broker)
	// fmt.Printf("Received Msg list:%v\n", msgs)
	// if len(msgs) != 5 {
	// 	t.Errorf("test schd store err,msg counter:%v", len(msgs))
	// }

	msgs = testc.startTestReceiptStore(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 3 {
		t.Errorf("test receipt store err,msg counter:%v", len(msgs))
	}

	msgs = testc.startTestIndexStore(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 4 {
		t.Errorf("test index store err,msg counter:%v", len(msgs))
	}

	msgs = testc.startTestBlockStore(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 3 {
		t.Errorf("test block store err,msg counter:%v", len(msgs))
	}

	defer ClearPath(basepath)
}

func TestStorage(t *testing.T) {
	basepath := "./case_storage"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_storage.yaml")

	testc := NewStorageTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	// testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTest(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 0 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestStorageQuery(t *testing.T) {
	basepath := "./case_storage"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_storage.yaml")

	testc := NewStorageTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTestQueryRpcBase(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 35 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestExecutor(t *testing.T) {
	basepath := "./case_executor"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_executor.yaml")

	testc := NewExecutorTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	// testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTestRpc(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 4 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestPoolAsL1(t *testing.T) {
	basepath := "./case_pool"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_pool.yaml")

	testc := NewPoolTest(basepath, true)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	mtypes.RunAsL1 = true

	msgs := testc.startTestAsL1(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 7 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestPoolAsL2(t *testing.T) {
	basepath := "./case_pool2"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_pool2.yaml")

	testc := NewPoolTest(basepath, false)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	mtypes.RunAsL1 = false

	msgs := testc.startTestAsL2(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 8 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestScheduler(t *testing.T) {
	basepath := "./cfg_scheduler"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_scheduler.yaml")

	arbi := NewMockArbitrator(basepath)
	actor.CreateActor("arbi", broker, []actor.Business{arbi}, []string{"arbi"}, 10, []string{""})

	exec := NewMockExecutor(basepath)
	actor.CreateActor("exec", broker, []actor.Business{exec}, []string{"exec"}, 10, []string{""})

	testc := NewSchedulerTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTest(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 5 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestArbitrator(t *testing.T) {
	basepath := "./cfg_arbitrator"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_arbitrator.yaml")

	testc := NewArbitratorTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTest(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 0 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestReceiptHashing(t *testing.T) {
	basepath := "./case_receipt-hashing"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_receipt-hashing.yaml")

	testc := NewReceiptHashingTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	// testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTest(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 2 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestGatewayTppTxBlock(t *testing.T) {
	basepath := "./case_gateway_tpp"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_gateway_tpp.yaml")

	testc := NewGatewayTppTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTest(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 3 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestGatewayTppLocalSingle(t *testing.T) {
	basepath := "./case_gateway_tpp"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_gateway_tpp.yaml")

	testc := NewGatewayTppTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTestLocalSingle(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 6 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestGatewayTppLocalBatch(t *testing.T) {
	basepath := "./case_gateway_tpp"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_gateway_tpp.yaml")

	testc := NewGatewayTppTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTestLocalBatch(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 5 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestCore(t *testing.T) {
	basepath := "./case_core"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_core.yaml")

	testc := NewMakeCoreTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTest(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 6 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestCoordinator(t *testing.T) {
	basepath := "./case_coordinator"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_coordinator.yaml")

	testc := NewCoordinatorTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	// testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTest(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 8 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}

func TestEthapi(t *testing.T) {
	basepath := "./case_eth_api"
	app, broker, _ := InitCfg(basepath, "../global.yaml", "../jet.yaml", "./cfg_eth_api.yaml")

	testc := NewEthApiTest(basepath)
	actor.CreateActor("sender", broker, []actor.Business{testc}, []string{"sender"}, 10, []string{""})
	testc.SetSender(actor.NewSendAdaptor(broker, rpc.GlobalRPCClient))
	StartSys(app, broker)

	msgs := testc.startTest(broker)
	fmt.Printf("Received Msg list:%v\n", msgs)
	if len(msgs) != 5 {
		t.Errorf("test err,msg counter:%v", len(msgs))
	}
	ClearPath(basepath)
}
