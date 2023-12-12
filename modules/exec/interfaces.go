package exec

import (
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	ccurl "github.com/arcology-network/concurrenturl"
	"github.com/arcology-network/concurrenturl/interfaces"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	adaptor "github.com/arcology-network/vm-adaptor/execution"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type SnapshotDict interface {
	Reset(snapshot *interfaces.Datastore)
	AddItem(precedingHash evmCommon.Hash, size int, snapshot *interfaces.Datastore)
	Query(precedings []*evmCommon.Hash) (*interfaces.Datastore, []*evmCommon.Hash)
}

type ExecutionImpl interface {
	Init(eu *adaptor.EU, url *ccurl.ConcurrentUrl)
	SetDB(db *interfaces.Datastore)
	Exec(sequence *cmntyp.ExecutingSequence, config *adaptor.Config, logger *actor.WorkerThreadLogger, gatherExeclog bool) (*exetyp.ExecutionResponse, error)
}
