package exec

import (
	eupk "github.com/arcology-network/eu"
	eucommon "github.com/arcology-network/eu/common"
	eushared "github.com/arcology-network/eu/shared"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	mtypes "github.com/arcology-network/main/types"
	stgcommitter "github.com/arcology-network/storage-committer/committer"
	"github.com/arcology-network/storage-committer/interfaces"
	"github.com/arcology-network/streamer/actor"
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
		// actor.MsgExecutingLogs,
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
				&eushared.EuResult{
					H: string(evmCommon.BytesToHash([]byte{1}).Bytes()),
				},
			},
			msg.Height,
		)
	}
	return nil
}

type mockSnapshotDict struct {
	base *interfaces.ReadOnlyStore
}

func (m *mockSnapshotDict) Reset(snapshot *interfaces.ReadOnlyStore) {
	m.base = snapshot
}

func (m *mockSnapshotDict) AddItem(precedingHash evmCommon.Hash, size int, snapshot *interfaces.ReadOnlyStore) {
}

func (m *mockSnapshotDict) Query(precedings []*evmCommon.Hash) (*interfaces.ReadOnlyStore, []*evmCommon.Hash) {
	if len(precedings) == 0 {
		return m.base, nil
	} else {
		return m.base, precedings
	}
}

type mockExecutionImpl struct {
}

func (m *mockExecutionImpl) Init(eu *eupk.EU, url *stgcommitter.StateCommitter) {}

func (m *mockExecutionImpl) SetDB(db *interfaces.ReadOnlyStore) {}

func (m *mockExecutionImpl) Exec(
	sequence *mtypes.ExecutingSequence,
	config *eucommon.Config,
	logger *actor.WorkerThreadLogger,
	gatherExeclog bool,
) (*exetyp.ExecutionResponse, error) {
	return &exetyp.ExecutionResponse{
		EuResults: []*eushared.EuResult{
			{},
		},
		Receipts: []*ethTypes.Receipt{
			{},
		},
		ExecutingLogs: []string{""},
	}, nil
}
