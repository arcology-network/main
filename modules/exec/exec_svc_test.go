package exec

import (
	"math/big"
	"testing"
	"time"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	ccdb "github.com/arcology-network/concurrenturl/storage"
	"github.com/arcology-network/streamer/actor"
	brokerpk "github.com/arcology-network/streamer/broker"
	intf "github.com/arcology-network/streamer/interface"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

func TestExecSvcBasic(t *testing.T) {
	setup(t)

	var response types.ExecutorResponses
	intf.Router.Call("executor-1", "ExecTxs", &actor.Message{
		Msgid:  common.GenerateUUID(),
		Height: 2,
		Data: &types.ExecutorRequest{
			Sequences: []*types.ExecutingSequence{
				{
					Msgs: []*types.StandardTransaction{
						{},
					},
					Parallel: true,
					Txids:    []uint32{1},
				},
			},
			Precedings: [][]*evmCommon.Hash{
				{},
			},
			PrecedingHash: []evmCommon.Hash{
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

	var response types.ExecutorResponses
	precedingHash := evmCommon.BytesToHash([]byte{1})
	intf.Router.Call("executor-1", "ExecTxs", &actor.Message{
		Msgid:  common.GenerateUUID(),
		Height: 2,
		Data: &types.ExecutorRequest{
			Sequences: []*types.ExecutingSequence{
				{
					Msgs: []*types.StandardTransaction{
						{},
					},
					Parallel: true,
					Txids:    []uint32{1},
				},
				{
					Msgs: []*types.StandardTransaction{
						{},
					},
					Parallel: true,
					Txids:    []uint32{2},
				},
			},
			Precedings: [][]*evmCommon.Hash{
				{},
				{
					&precedingHash,
				},
			},
			PrecedingHash: []evmCommon.Hash{
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

func setup(tb testing.TB) (*brokerpk.StatefulStreamer, *mockWorker) {
	log.InitLog("exec-svc.log", "./log.toml", "tester", "tester", 0)
	// config.MainConfig = &config.Monaco{
	// 	ChainId: new(big.Int),
	// }
	broker := brokerpk.NewStatefulStreamer()

	intf.RPCCreator = func(serviceAddr, basepath string, zkAddrs []string, rcvrs, fns []interface{}) {}
	rpc := NewRpcService(4, "rpc").(*RpcService)
	rpcActor := actor.NewActorEx("rpc", broker, rpc)
	rpcActor.Connect(brokerpk.NewDisjunctions(rpcActor, 1))
	intf.Router.Register("executor-1", rpc, "rpc-server-addr", "zk-server-addr")

	execImpl := NewExecutor(1, "exec-impl").(*Executor)
	execImpl.snapshotDict = &mockSnapshotDict{}
	baseWorker := actor.NewHeightController()
	baseWorker.Next(actor.NewFSMController()).EndWith(execImpl)
	baseWorker.OnStart()
	// execImpl.eus[0] = &mockExecutionImpl{}
	execImplActor := actor.NewActorEx("exec-impl", broker, baseWorker)
	execImplActor.Connect(brokerpk.NewDisjunctions(execImplActor, 1))

	mock := newMockWorker(4, "mock").(*mockWorker)
	mockActor := actor.NewActorEx("mock", broker, mock)
	mockActor.Connect(brokerpk.NewDisjunctions(mockActor, 1))
	broker.Serve()

	// var db interfaces.Datastore = cachedstorage.NewDataStore(
	// 	nil,
	// 	cachedstorage.NewCachePolicy(math.MaxUint64, 1),
	// 	storage.NewReadonlyRpcClient(),
	// 	// func(v interface{}) []byte { return urltyp.ToBytes(v) },
	// 	// func(bytes []byte) interface{} { return urltyp.FromBytes(bytes) },

	// 	// func(v interface{}) []byte {
	// 	// 	return ccdb.Codec{}.Encode(v)
	// 	// },
	// 	// func(bytes []byte) interface{} {
	// 	// 	return ccdb.Codec{}.Decode(bytes)
	// 	// },

	// 	// cachedstorage.NotQueryRpc,
	// 	ccdb.Rlp{}.Encode,
	// 	ccdb.Rlp{}.Decode,
	// )
	db := ccdb.NewParallelEthMemDataStore()

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
					Data: &types.ParentInfo{},
				},
			},
		},
		2,
	)
	return broker, mock
}
