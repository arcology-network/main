package exec

import (
	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	ccurl "github.com/arcology-network/concurrenturl/v2"
	urlcmn "github.com/arcology-network/concurrenturl/v2/common"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	adaptor "github.com/arcology-network/vm-adaptor/evm"
)

type SnapshotDict interface {
	Reset(snapshot *urlcmn.DatastoreInterface)
	AddItem(precedingHash ethcmn.Hash, size int, snapshot *urlcmn.DatastoreInterface)
	Query(precedings []*ethcmn.Hash) (*urlcmn.DatastoreInterface, []*ethcmn.Hash)
}

type ExecutionImpl interface {
	Init(eu *adaptor.EUV2, url *ccurl.ConcurrentUrl)
	SetDB(db *urlcmn.DatastoreInterface)
	Exec(sequence *cmntyp.ExecutingSequence, config *adaptor.Config, logger *actor.WorkerThreadLogger, gatherExeclog bool) (*exetyp.ExecutionResponse, error)
}
