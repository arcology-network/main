package scheduler

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"

	cmncmn "github.com/arcology-network/common-lib/common"
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
	"github.com/arcology-network/component-lib/log"
	"github.com/arcology-network/component-lib/streamer"
	"github.com/arcology-network/main/modules/storage"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

func TestSchedulerEmptyBlock(t *testing.T) {
	schd := setup(t)

	b1 := &block{
		timestamp: new(big.Int).SetUint64(100),
		msgs:      []*cmntyp.StandardMessage{},
		height:    1,
	}
	schd.OnMessageArrived(b1.getMsgs())

	time.Sleep(1 * time.Second)
}

func TestSchedulerTransferOnly(t *testing.T) {
	schd := setup(t)

	runBlock(
		schd,
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{101}),
			evmCommon.BytesToAddress([]byte{102}),
		},
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{11}),
			evmCommon.BytesToAddress([]byte{12}),
		},
		[]evmCommon.Address{},
		[2][]evmCommon.Address{{}, {}},
		[]*cmntyp.ExecutorResponses{
			{
				// DfCalls: []*cmntyp.DeferCall{nil, nil},
				HashList: []evmCommon.Hash{
					evmCommon.BytesToHash([]byte{1}),
					evmCommon.BytesToHash([]byte{2}),
				},
				StatusList:  []uint64{1, 1},
				GasUsedList: []uint64{21000, 21000},
			},
		},
		[]*cmntyp.ArbitratorResponse{{}, {}},
		map[string]interface{}{
			actor.MsgSchdState: &storage.SchdState{},
			actor.MsgInclusive: &cmntyp.InclusiveList{
				HashList:   []*evmCommon.Hash{nil, nil},
				Successful: []bool{true, true},
			},
			// actor.MsgSpawnedRelations: []*cmntyp.SpawnedRelation{},
		},
	)
}

func TestSchedulerMixedTxs(t *testing.T) {
	schd := setup(t)

	runBlock(
		schd,
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{101}),
			evmCommon.BytesToAddress([]byte{102}),
			evmCommon.BytesToAddress([]byte{103}),
			evmCommon.BytesToAddress([]byte{104}),
		},
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{21}),
			evmCommon.BytesToAddress([]byte{22}),
			evmCommon.BytesToAddress([]byte{23}),
			evmCommon.BytesToAddress([]byte{24}),
		},
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{23}),
			evmCommon.BytesToAddress([]byte{24}),
		},
		[2][]evmCommon.Address{{}, {}},
		[]*cmntyp.ExecutorResponses{
			{
				// DfCalls: []*cmntyp.DeferCall{nil, nil},
				HashList: []evmCommon.Hash{
					evmCommon.BytesToHash([]byte{1}),
					evmCommon.BytesToHash([]byte{2}),
				},
				StatusList:  []uint64{1, 1},
				GasUsedList: []uint64{21000, 21000},
			},
			{
				// DfCalls: []*cmntyp.DeferCall{nil, nil},
				HashList: []evmCommon.Hash{
					evmCommon.BytesToHash([]byte{3}),
					evmCommon.BytesToHash([]byte{4}),
				},
				StatusList:  []uint64{1, 1},
				GasUsedList: []uint64{50000, 50000},
			},
		},
		[]*cmntyp.ArbitratorResponse{{}, {}},
		map[string]interface{}{
			actor.MsgSchdState: &storage.SchdState{},
			actor.MsgInclusive: &cmntyp.InclusiveList{
				HashList:   []*evmCommon.Hash{nil, nil, nil, nil},
				Successful: []bool{true, true, true, true},
			},
			// actor.MsgSpawnedRelations: []*cmntyp.SpawnedRelation{},
		},
	)
}

func TestSchedulerContractWithDefer(t *testing.T) {
	schd := setup(t)

	spawnedHash := sha256.Sum256(cmncmn.Flatten([][]byte{
		evmCommon.BytesToHash([]byte{3}).Bytes(),
		evmCommon.BytesToHash([]byte{4}).Bytes(),
	}))
	// seqId := sha256.Sum256(encoding.Byteset([][]byte{spawnedHash[:]}).Encode())
	runBlock(
		schd,
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{101}),
			evmCommon.BytesToAddress([]byte{102}),
			evmCommon.BytesToAddress([]byte{103}),
			evmCommon.BytesToAddress([]byte{104}),
		},
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{31}),
			evmCommon.BytesToAddress([]byte{32}),
			evmCommon.BytesToAddress([]byte{33}),
			evmCommon.BytesToAddress([]byte{33}),
		},
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{31}),
			evmCommon.BytesToAddress([]byte{32}),
			evmCommon.BytesToAddress([]byte{33}),
		},
		[2][]evmCommon.Address{{}, {}},
		[]*cmntyp.ExecutorResponses{
			{
				// DfCalls: []*cmntyp.DeferCall{
				// 	nil,
				// 	nil,
				// 	{
				// 		DeferID:         "deferid",
				// 		ContractAddress: cmntyp.Address(evmCommon.BytesToAddress([]byte{33}).Bytes()),
				// 		Signature:       string([]byte{1, 2, 3, 4}),
				// 	},
				// 	{
				// 		DeferID:         "deferid",
				// 		ContractAddress: cmntyp.Address(evmCommon.BytesToAddress([]byte{33}).Bytes()),
				// 		Signature:       string([]byte{1, 2, 3, 4}),
				// 	},
				// },
				HashList: []evmCommon.Hash{
					evmCommon.BytesToHash([]byte{1}),
					evmCommon.BytesToHash([]byte{2}),
					evmCommon.BytesToHash([]byte{3}),
					evmCommon.BytesToHash([]byte{4}),
				},
				StatusList:  []uint64{1, 1, 1, 1},
				GasUsedList: []uint64{50000, 50000, 50000, 50000},
			},
			{
				// DfCalls:        []*cmntyp.DeferCall{nil},
				HashList:    []evmCommon.Hash{evmCommon.BytesToHash(spawnedHash[:])},
				StatusList:  []uint64{1},
				GasUsedList: []uint64{50000},
				// RelationKeys:   []evmCommon.Hash{evmCommon.BytesToHash(seqId[:])},
				// RelationSizes:  []uint64{1},
				// RelationValues: []evmCommon.Hash{evmCommon.BytesToHash(spawnedHash[:])},
			},
		},
		[]*cmntyp.ArbitratorResponse{{}, {}},
		map[string]interface{}{
			actor.MsgSchdState: &storage.SchdState{},
			actor.MsgInclusive: &cmntyp.InclusiveList{
				HashList:   []*evmCommon.Hash{nil, nil, nil, nil, nil},
				Successful: []bool{true, true, true, true, true},
			},
			// actor.MsgSpawnedRelations: []*cmntyp.SpawnedRelation{nil, nil},
		},
	)
}

func TestSchedulerContractWithConfliction(t *testing.T) {
	schd := setup(t)

	conflictHash := evmCommon.BytesToHash([]byte{2})
	runBlock(
		schd,
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{101}),
			evmCommon.BytesToAddress([]byte{102}),
		},
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{41}),
			evmCommon.BytesToAddress([]byte{42}),
		},
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{41}),
			evmCommon.BytesToAddress([]byte{42}),
		},
		[2][]evmCommon.Address{{}, {}},
		[]*cmntyp.ExecutorResponses{
			{
				// DfCalls: []*cmntyp.DeferCall{nil, nil},
				HashList: []evmCommon.Hash{
					evmCommon.BytesToHash([]byte{1}),
					evmCommon.BytesToHash([]byte{2}),
				},
				StatusList:  []uint64{1, 1},
				GasUsedList: []uint64{50000, 50000},
			},
		},
		[]*cmntyp.ArbitratorResponse{
			{
				ConflictedList: []*evmCommon.Hash{&conflictHash},
				CPairLeft:      []uint32{256},
				CPairRight:     []uint32{512},
			},
		},
		map[string]interface{}{
			actor.MsgSchdState: &storage.SchdState{
				ConflictionLefts:  []evmCommon.Address{{}},
				ConflictionRights: []evmCommon.Address{{}},
			},
			actor.MsgInclusive: &cmntyp.InclusiveList{
				HashList:   []*evmCommon.Hash{nil, nil},
				Successful: []bool{true, false},
			},
			// actor.MsgSpawnedRelations: []*cmntyp.SpawnedRelation{},
		},
	)
}

func TestSchedulerSequentialTxs(t *testing.T) {
	schd := setup(t)

	// hashes := []evmCommon.Hash{
	// 	evmCommon.BytesToHash([]byte{1}),
	// 	evmCommon.BytesToHash([]byte{2}),
	// }
	// seqId := sha256.Sum256(encoding.Byteset([][]byte{
	// 	hashes[0].Bytes(),
	// 	hashes[1].Bytes(),
	// }).Encode())
	runBlock(
		schd,
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{101}),
			evmCommon.BytesToAddress([]byte{102}),
		},
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{51}),
			evmCommon.BytesToAddress([]byte{51}),
		},
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{51}),
		},
		[2][]evmCommon.Address{
			{evmCommon.BytesToAddress([]byte{51})},
			{evmCommon.BytesToAddress([]byte{51})},
		},
		[]*cmntyp.ExecutorResponses{
			{
				// DfCalls: []*cmntyp.DeferCall{nil, nil},
				HashList: []evmCommon.Hash{
					evmCommon.BytesToHash([]byte{1}),
					evmCommon.BytesToHash([]byte{2}),
				},
				StatusList:  []uint64{1, 1},
				GasUsedList: []uint64{50000, 50000},
				// SpawnedKeys: []evmCommon.Hash{
				// 	evmCommon.BytesToHash([]byte{1}),
				// },
				// SpawnedTxs: []evmCommon.Hash{
				// 	evmCommon.BytesToHash([]byte{1, 100}),
				// },
				// RelationKeys:  []evmCommon.Hash{evmCommon.BytesToHash(seqId[:])},
				// RelationSizes: []uint64{3},
				// RelationValues: []evmCommon.Hash{
				// 	evmCommon.BytesToHash([]byte{1}),
				// 	evmCommon.BytesToHash([]byte{1, 100}),
				// 	evmCommon.BytesToHash([]byte{2}),
				// },
				// TxidsHash: []evmCommon.Hash{
				// 	evmCommon.BytesToHash([]byte{1, 100}),
				// },
				// TxidsId: []uint32{257},
				// TxidsAddress: []evmCommon.Address{
				// 	evmCommon.BytesToAddress([]byte{51}),
				// },
			},
		},
		[]*cmntyp.ArbitratorResponse{},
		map[string]interface{}{
			actor.MsgSchdState: &storage.SchdState{},
			actor.MsgInclusive: &cmntyp.InclusiveList{
				HashList:   []*evmCommon.Hash{nil, nil, nil},
				Successful: []bool{true, true, true},
			},
			// actor.MsgSpawnedRelations: []*cmntyp.SpawnedRelation{nil},
		},
	)
}

func TestSchedulerConflictionInDefer(t *testing.T) {
	schd := setup(t)

	txHashes := []evmCommon.Hash{
		evmCommon.BytesToHash([]byte{1}),
		evmCommon.BytesToHash([]byte{2}),
		evmCommon.BytesToHash([]byte{3}),
		evmCommon.BytesToHash([]byte{4}),
	}
	spawnedHashes := [][32]byte{
		sha256.Sum256(cmncmn.Flatten([][]byte{txHashes[0].Bytes()})),
		sha256.Sum256(cmncmn.Flatten([][]byte{txHashes[2].Bytes(), txHashes[3].Bytes()})),
	}
	txHashes = append(txHashes, []evmCommon.Hash{
		evmCommon.BytesToHash(spawnedHashes[0][:]),
		evmCommon.BytesToHash(spawnedHashes[1][:]),
	}...)
	// seqIds := [][32]byte{
	// 	sha256.Sum256(encoding.Byteset([][]byte{spawnedHashes[0][:]}).Encode()),
	// 	sha256.Sum256(encoding.Byteset([][]byte{spawnedHashes[1][:]}).Encode()),
	// }
	runBlock(
		schd,
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{101}),
			evmCommon.BytesToAddress([]byte{102}),
			evmCommon.BytesToAddress([]byte{103}),
			evmCommon.BytesToAddress([]byte{104}),
		},
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{61}),
			evmCommon.BytesToAddress([]byte{61}),
			evmCommon.BytesToAddress([]byte{62}),
			evmCommon.BytesToAddress([]byte{62}),
		},
		[]evmCommon.Address{
			evmCommon.BytesToAddress([]byte{61}),
			evmCommon.BytesToAddress([]byte{62}),
		},
		[2][]evmCommon.Address{{}, {}},
		[]*cmntyp.ExecutorResponses{
			{
				// DfCalls: []*cmntyp.DeferCall{
				// 	{
				// 		DeferID:         "deferid1",
				// 		ContractAddress: cmntyp.Address(evmCommon.BytesToAddress([]byte{61}).Bytes()),
				// 		Signature:       string([]byte{1, 2, 3, 4}),
				// 	},
				// 	{
				// 		DeferID:         "deferid1",
				// 		ContractAddress: cmntyp.Address(evmCommon.BytesToAddress([]byte{61}).Bytes()),
				// 		Signature:       string([]byte{1, 2, 3, 4}),
				// 	},
				// 	{
				// 		DeferID:         "deferid2",
				// 		ContractAddress: cmntyp.Address(evmCommon.BytesToAddress([]byte{62}).Bytes()),
				// 		Signature:       string([]byte{5, 6, 7, 8}),
				// 	},
				// 	{
				// 		DeferID:         "deferid2",
				// 		ContractAddress: cmntyp.Address(evmCommon.BytesToAddress([]byte{62}).Bytes()),
				// 		Signature:       string([]byte{5, 6, 7, 8}),
				// 	},
				// },
				HashList:    txHashes[:4],
				StatusList:  []uint64{1, 1, 1, 1},
				GasUsedList: []uint64{50000, 50000, 50000, 50000},
			},
			{
				// DfCalls:        []*cmntyp.DeferCall{nil, nil},
				HashList:    []evmCommon.Hash{txHashes[4], txHashes[5]},
				StatusList:  []uint64{1, 1},
				GasUsedList: []uint64{50000, 50000},
				// RelationKeys:   []evmCommon.Hash{evmCommon.BytesToHash(seqIds[0][:]), evmCommon.BytesToHash(seqIds[1][:])},
				// RelationSizes:  []uint64{1, 1},
				// RelationValues: []evmCommon.Hash{txHashes[4], txHashes[5]},
			},
		},
		[]*cmntyp.ArbitratorResponse{
			{
				ConflictedList: []*evmCommon.Hash{&txHashes[1]},
				CPairLeft:      []uint32{256},
				CPairRight:     []uint32{512},
			},
			{},
			{
				ConflictedList: []*evmCommon.Hash{&txHashes[2], &txHashes[3], &txHashes[5]},
				CPairLeft:      []uint32{257},
				CPairRight:     []uint32{768},
			},
		},
		map[string]interface{}{
			actor.MsgSchdState: &storage.SchdState{
				ConflictionLefts:  []evmCommon.Address{{}, {}},
				ConflictionRights: []evmCommon.Address{{}, {}},
			},
			actor.MsgInclusive: &cmntyp.InclusiveList{
				HashList:   cmncmn.ToReferencedSlice(txHashes),
				Successful: []bool{true, false, false, false, true, false},
			},
			// actor.MsgSpawnedRelations: []*cmntyp.SpawnedRelation{nil, nil, nil},
		},
	)
}

type mockConsumer struct {
	tb              testing.TB
	expectedResults map[string]interface{}
}

func (mock *mockConsumer) Consume(data interface{}) {
	mock.tb.Log(data, data.(streamer.Aggregated).Data.(*actor.Message).Data)
	mock.check(data)
}

func (mock *mockConsumer) check(data interface{}) {
	msgName := data.(streamer.Aggregated).Name
	_, ok := mock.expectedResults[msgName]
	if !ok {
		return
	}

	msg := data.(streamer.Aggregated).Data.(*actor.Message).Data
	switch msgName {
	case actor.MsgSchdState:
		got := msg.(*storage.SchdState)
		expected := mock.expectedResults[msgName].(*storage.SchdState)
		if len(expected.NewContracts) != len(got.NewContracts) ||
			len(expected.ConflictionLefts) != len(got.ConflictionLefts) {
			panic(fmt.Sprintf("check %s failed, expected %v, got %v", msgName, expected, got))
		}
	case actor.MsgInclusive:
		got := msg.(*cmntyp.InclusiveList)
		expected := mock.expectedResults[msgName].(*cmntyp.InclusiveList)
		if len(expected.HashList) != len(got.HashList) {
			panic(fmt.Sprintf("check %s failed, expected %v, got %v", msgName, expected, got))
		}
		if len(expected.HashList) > 0 && expected.HashList[0] != nil {
			expectedDict := make(map[evmCommon.Hash]bool)
			gotDict := make(map[evmCommon.Hash]bool)
			for i := range expected.HashList {
				expectedDict[*expected.HashList[i]] = expected.Successful[i]
				gotDict[*got.HashList[i]] = got.Successful[i]
			}
			if !reflect.DeepEqual(expectedDict, gotDict) {
				panic(fmt.Sprintf("check %s failed, expected %v, got %v", msgName, expected, got))
			}
		} else {
			if !reflect.DeepEqual(expected.Successful, got.Successful) {
				panic(fmt.Sprintf("check %s failed, expected %v, got %v", msgName, expected, got))
			}
		}
		// case actor.MsgSpawnedRelations:
		// 	got := msg.([]*cmntyp.SpawnedRelation)
		// 	expected := mock.expectedResults[msgName].([]*cmntyp.SpawnedRelation)
		// 	if len(got) != len(expected) {
		// 		panic(fmt.Sprintf("check %s failed, expected %v, got %v", msgName, expected, got))
		// 	}
	}
}

var (
	consumer *mockConsumer
)

func setup(tb testing.TB) *Scheduler {
	log.InitLog("scheduler-test.log", "./log.toml", "tester", "tester", 0)

	intf.RPCCreator = func(serviceAddr, basepath string, zkAddrs []string, rcvrs, fns []interface{}) {}
	intf.Router.Register("schdstore", &schdStoreMock{}, "rpc-server-addr", "zk-server-addr")
	executor.tb = tb
	intf.Router.Register("executor-1", &executor, "rpc-server-addr", "zk-server-addr")
	arbitrator.tb = tb
	intf.Router.Register("arbitrator", &arbitrator, "rpc-server-addr", "zk-server-addr")
	intf.Router.SetAvailableServices([]string{"executor-1"})

	broker := streamer.NewStatefulStreamer()
	consumer = &mockConsumer{tb: tb}
	broker.RegisterProducer(streamer.NewDefaultProducer(
		"scheduler",
		[]string{
			actor.MsgInclusive,
			actor.MsgExecTime,
			actor.MsgSpawnedRelations,
			actor.MsgSchdState,
		},
		[]int{1, 1, 1, 1},
	))
	broker.RegisterConsumer(streamer.NewDefaultConsumer(
		"mock-consumer",
		[]string{
			actor.MsgInclusive,
			actor.MsgExecTime,
			actor.MsgSpawnedRelations,
			actor.MsgSchdState,
		},
		streamer.NewDisjunctions(consumer, 1),
	))
	broker.Serve()

	schd := NewScheduler(4, "tester").(*Scheduler)
	schd.Init("scheduler", broker)
	schd.Config(map[string]interface{}{
		"batch_size":    float64(100),
		"parallelism":   float64(4),
		"conflict_file": "conflict_file",
	})
	return schd
}

func runBlock(
	schd *Scheduler,
	msgFroms, msgTos []evmCommon.Address,
	contracts []evmCommon.Address,
	conflicts [2][]evmCommon.Address,
	execResponse []*cmntyp.ExecutorResponses,
	arbResponse []*cmntyp.ArbitratorResponse,
	expectedResults map[string]interface{},
) {
	msgs := make([]core.Message, len(msgFroms))
	for i := range msgFroms {
		msgs[i] = core.NewMessage(
			msgFroms[i],
			&msgTos[i],
			1,
			new(big.Int).SetUint64(10),
			100000,
			new(big.Int).SetUint64(10),
			[]byte{},
			nil,
			false,
		)
	}

	stdMsgs := make([]*cmntyp.StandardMessage, len(msgs))
	for i := range msgs {
		stdMsgs[i] = &cmntyp.StandardMessage{
			TxHash: evmCommon.BytesToHash([]byte{byte(i + 1)}),
			Native: &msgs[i],
		}
	}

	b := &block{
		timestamp:       new(big.Int).SetUint64(100),
		msgs:            stdMsgs,
		height:          1,
		execResponse:    execResponse,
		arbResponse:     arbResponse,
		expectedResults: expectedResults,
	}

	for _, contract := range contracts {
		schd.contractDict[contract] = struct{}{}
	}
	if len(conflicts[0]) > 0 {
		schd.schdEngine.Update(
			cmncmn.ToReferencedSlice(conflicts[0]),
			cmncmn.ToReferencedSlice(conflicts[1]),
		)
	}

	schd.OnMessageArrived(b.getMsgs())
	time.Sleep(1 * time.Second)
}

type block struct {
	timestamp *big.Int
	msgs      []*cmntyp.StandardMessage
	height    uint64

	execResponse    []*cmntyp.ExecutorResponses
	arbResponse     []*cmntyp.ArbitratorResponse
	expectedResults map[string]interface{}
}

func (b *block) getMsgs() []*actor.Message {
	executor.callTime = 0
	executor.response = b.execResponse
	arbitrator.callTime = 0
	arbitrator.response = b.arbResponse
	consumer.expectedResults = b.expectedResults

	return []*actor.Message{
		{
			Name: actor.MsgBlockStart,
			Data: &actor.BlockStart{
				Timestamp: b.timestamp,
			},
			Height: b.height,
		},
		{
			Name: actor.MsgMessagersReaped,
			Data: cmntyp.SendingStandardMessages{
				Data: cmntyp.StandardMessages(b.msgs).EncodeToBytes(),
			},
			Height: b.height,
		},
	}
}
