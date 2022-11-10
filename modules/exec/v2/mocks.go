package exec

import (
	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	ethtyp "github.com/arcology-network/3rd-party/eth/types"
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	ccurl "github.com/arcology-network/concurrenturl/v2"
	urlcmn "github.com/arcology-network/concurrenturl/v2/common"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	adaptor "github.com/arcology-network/vm-adaptor/evm"
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
				&cmntyp.EuResult{
					H: string(ethcmn.BytesToHash([]byte{1}).Bytes()),
				},
			},
			msg.Height,
		)
	}
	return nil
}

type mockSnapshotDict struct {
	base *urlcmn.DatastoreInterface
}

func (m *mockSnapshotDict) Reset(snapshot *urlcmn.DatastoreInterface) {
	m.base = snapshot
}

func (m *mockSnapshotDict) AddItem(precedingHash ethcmn.Hash, size int, snapshot *urlcmn.DatastoreInterface) {
}

func (m *mockSnapshotDict) Query(precedings []*ethcmn.Hash) (*urlcmn.DatastoreInterface, []*ethcmn.Hash) {
	if len(precedings) == 0 {
		return m.base, nil
	} else {
		return m.base, precedings
	}
}

type mockExecutionImpl struct {
}

func (m *mockExecutionImpl) Init(eu *adaptor.EUV2, url *ccurl.ConcurrentUrl) {}

func (m *mockExecutionImpl) SetDB(db *urlcmn.DatastoreInterface) {}

func (m *mockExecutionImpl) Exec(
	sequence *cmntyp.ExecutingSequence,
	config *adaptor.Config,
	logger *actor.WorkerThreadLogger,
	gatherExeclog bool,
) (*exetyp.ExecutionResponse, error) {
	return &exetyp.ExecutionResponse{
		EuResults: []*cmntyp.EuResult{
			{},
		},
		Receipts: []*ethtyp.Receipt{
			{},
		},
		ExecutingLogs: []string{""},
	}, nil
}
