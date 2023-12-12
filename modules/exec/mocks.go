package exec

import (
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	ccurl "github.com/arcology-network/concurrenturl"
	"github.com/arcology-network/concurrenturl/interfaces"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	adaptor "github.com/arcology-network/vm-adaptor/execution"
	evmCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

type mockWorker struct {
	actor.WorkerThread
}

func newMockWorker(concurrency int, groupId string) actor.IWorkerEx {
	m := &mockWorker{}
	m.Set(concurrency, groupId)
	return m
}

func (m *mockWorker) Inputs() ([]string, bool) {
	return []string{
		actor.MsgPrecedingList,
		actor.MsgExecuted,
		actor.MsgReceipts,
		actor.MsgReceiptHashList,
		actor.MsgExecutingLogs,
		// Kafka uploader.
		actor.MsgTxAccessRecords,
	}, false
}

func (m *mockWorker) Outputs() map[string]int {
	return map[string]int{
		actor.CombinedName(actor.MsgApcHandle, actor.MsgCached):      1,
		actor.CombinedName(actor.MsgBlockStart, actor.MsgParentInfo): 1,
		actor.MsgPrecedingsEuresult:                                  1,
		actor.MsgSelectedExecuted:                                    1,
	}
}

func (m *mockWorker) OnStart() {}

func (m *mockWorker) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch msg.Name {
	case actor.MsgPrecedingList:
		m.MsgBroker.Send(
			actor.MsgPrecedingsEuresult,
			[]interface{}{
				&types.EuResult{
					H: string(evmCommon.BytesToHash([]byte{1}).Bytes()),
				},
			},
			msg.Height,
		)
	}
	return nil
}

type mockSnapshotDict struct {
	base *interfaces.Datastore
}

func (m *mockSnapshotDict) Reset(snapshot *interfaces.Datastore) {
	m.base = snapshot
}

func (m *mockSnapshotDict) AddItem(precedingHash evmCommon.Hash, size int, snapshot *interfaces.Datastore) {
}

func (m *mockSnapshotDict) Query(precedings []*evmCommon.Hash) (*interfaces.Datastore, []*evmCommon.Hash) {
	if len(precedings) == 0 {
		return m.base, nil
	} else {
		return m.base, precedings
	}
}

type mockExecutionImpl struct {
}

func (m *mockExecutionImpl) Init(eu *adaptor.EU, url *ccurl.ConcurrentUrl) {}

func (m *mockExecutionImpl) SetDB(db *interfaces.Datastore) {}

func (m *mockExecutionImpl) Exec(
	sequence *types.ExecutingSequence,
	config *adaptor.Config,
	logger *actor.WorkerThreadLogger,
	gatherExeclog bool,
) (*exetyp.ExecutionResponse, error) {
	return &exetyp.ExecutionResponse{
		EuResults: []*types.EuResult{
			{},
		},
		Receipts: []*ethTypes.Receipt{
			{},
		},
		ExecutingLogs: []string{""},
	}, nil
}
