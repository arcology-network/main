package exec

import (
	"math"
	"math/big"
	"testing"
	"time"

	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	cachedstorage "github.com/arcology-network/common-lib/cachedstorage"
	cmncmn "github.com/arcology-network/common-lib/common"
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/config"
	intf "github.com/arcology-network/component-lib/interface"
	"github.com/arcology-network/component-lib/log"
	"github.com/arcology-network/component-lib/storage"
	"github.com/arcology-network/component-lib/streamer"
	urlcmn "github.com/arcology-network/concurrenturl/v2/common"
	urltyp "github.com/arcology-network/concurrenturl/v2/type"
)

func TestExecSvcBasic(t *testing.T) {
	setup(t)

	var response cmntyp.ExecutorResponses
	intf.Router.Call("executor-1", "ExecTxs", &actor.Message{
		Msgid:  cmncmn.GenerateUUID(),
		Height: 2,
		Data: &cmntyp.ExecutorRequest{
			Sequences: []*cmntyp.ExecutingSequence{
				{
					Msgs: []*cmntyp.StandardMessage{
						{},
					},
					Parallel: true,
					Txids:    []uint32{1},
				},
			},
			Precedings: [][]*ethcmn.Hash{
				{},
			},
			PrecedingHash: []ethcmn.Hash{
				{},
			},
			Timestamp:   new(big.Int),
			Parallelism: 1,
			Debug:       false,
		},
	}, &response)

	time.Sleep(time.Second)
}

func TestExecSvcMakeSnapshot(t *testing.T) {
	_, mock := setup(t)

	var response cmntyp.ExecutorResponses
	precedingHash := ethcmn.BytesToHash([]byte{1})
	intf.Router.Call("executor-1", "ExecTxs", &actor.Message{
		Msgid:  cmncmn.GenerateUUID(),
		Height: 2,
		Data: &cmntyp.ExecutorRequest{
			Sequences: []*cmntyp.ExecutingSequence{
				{
					Msgs: []*cmntyp.StandardMessage{
						{},
					},
					Parallel: true,
					Txids:    []uint32{1},
				},
				{
					Msgs: []*cmntyp.StandardMessage{
						{},
					},
					Parallel: true,
					Txids:    []uint32{2},
				},
			},
			Precedings: [][]*ethcmn.Hash{
				{},
				{
					&precedingHash,
				},
			},
			PrecedingHash: []ethcmn.Hash{
				{},
				{},
			},
			Timestamp:   new(big.Int),
			Parallelism: 1,
			Debug:       false,
		},
	}, &response)

	mock.MsgBroker.Send(
		actor.MsgSelectedExecuted,
		nil,
		2,
	)

	time.Sleep(time.Second)
}

func setup(tb testing.TB) (*streamer.StatefulStreamer, *mockWorker) {
	log.InitLog("exec-svc.log", "./log.toml", "tester", "tester", 0)
	config.MainConfig = &config.Monaco{
		ChainId: new(big.Int),
	}
	broker := streamer.NewStatefulStreamer()

	intf.RPCCreator = func(serviceAddr, basepath string, zkAddrs []string, rcvrs, fns []interface{}) {}
	rpc := NewRpcService(4, "rpc").(*RpcService)
	rpcActor := actor.NewActorEx("rpc", broker, rpc)
	rpcActor.Connect(streamer.NewDisjunctions(rpcActor, 1))
	intf.Router.Register("executor-1", rpc, "rpc-server-addr", "zk-server-addr")

	execImpl := NewExecutor(1, "exec-impl").(*Executor)
	execImpl.snapshotDict = &mockSnapshotDict{}
	baseWorker := actor.NewHeightController()
	baseWorker.Next(actor.NewFSMController()).EndWith(execImpl)
	baseWorker.OnStart()
	execImpl.eus[0] = &mockExecutionImpl{}
	execImplActor := actor.NewActorEx("exec-impl", broker, baseWorker)
	execImplActor.Connect(streamer.NewDisjunctions(execImplActor, 1))

	mock := newMockWorker(4, "mock").(*mockWorker)
	mockActor := actor.NewActorEx("mock", broker, mock)
	mockActor.Connect(streamer.NewDisjunctions(mockActor, 1))
	broker.Serve()

	var db urlcmn.DatastoreInterface = cachedstorage.NewDataStore(
		nil,
		cachedstorage.NewCachePolicy(math.MaxUint64, 1),
		storage.NewReadonlyRpcClient(),
		func(v interface{}) []byte { return urltyp.ToBytes(v) },
		func(bytes []byte) interface{} { return urltyp.FromBytes(bytes) },
		cachedstorage.NotQueryRpc,
	)

	mock.MsgBroker.Send(
		actor.CombinedName(actor.MsgApcHandle, actor.MsgCached),
		&actor.CombinerElements{
			Msgs: map[string]*actor.Message{
				actor.MsgApcHandle: {
					Data: &db,
				},
				actor.MsgCached: {},
			},
		},
		1,
	)
	mock.MsgBroker.Send(
		actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo),
		&actor.CombinerElements{
			Msgs: map[string]*actor.Message{
				actor.MsgBlockStart: {
					Data: &actor.BlockStart{},
				},
				actor.MsgParentInfo: {
					Data: &cmntyp.ParentInfo{},
				},
			},
		},
		2,
	)
	return broker, mock
}
