package exec

import (
	eupk "github.com/arcology-network/eu"

	cmntyp "github.com/arcology-network/common-lib/types"
	ccurl "github.com/arcology-network/concurrenturl"
	"github.com/arcology-network/concurrenturl/interfaces"
	eucommon "github.com/arcology-network/eu/common"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type SnapshotDict interface {
	Reset(snapshot *interfaces.Datastore)
	AddItem(precedingHash evmCommon.Hash, size int, snapshot *interfaces.Datastore)
	Query(precedings []*evmCommon.Hash) (*interfaces.Datastore, []*evmCommon.Hash)
}

type ExecutionImpl interface {
	Init(eu *eupk.EU, url *ccurl.StateCommitter)
	SetDB(db *interfaces.Datastore)
	Exec(sequence *cmntyp.ExecutingSequence, config *eucommon.Config, logger *actor.WorkerThreadLogger, gatherExeclog bool) (*exetyp.ExecutionResponse, error)
}
